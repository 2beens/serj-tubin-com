package file_box

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

var (
	ErrFolderNotFound = errors.New("folder not found")
	ErrFileNotFound   = errors.New("file not found")
	ErrFolderExists   = errors.New("folder already exists")
)

type DiskApi struct {
	rootPath string
	root     *Folder
	mutex    sync.RWMutex
}

func NewDiskApi(rootPath string) (*DiskApi, error) {
	if rootPath == "" {
		return nil, errors.New("root path cannot be empty")
	}
	root, err := getRootFolder(context.Background(), rootPath)
	if err != nil {
		return nil, fmt.Errorf("get root folder: %w", err)
	}
	return &DiskApi{
		rootPath: rootPath,
		root:     root,
	}, nil
}

func (da *DiskApi) getFolder(parent *Folder, id int64) *Folder {
	if id == parent.Id {
		return parent
	}
	for _, f := range parent.Subfolders {
		if sf := da.getFolder(f, id); sf != nil {
			return sf
		}
	}
	return nil
}

func (da *DiskApi) getFile(id int64) (*File, *Folder) {
	for _, f := range da.root.Files {
		if f.Id == id {
			return f, da.root
		}
	}
	for _, f := range da.root.Subfolders {
		if file, parent := da.getFileRecursive(id, f); file != nil {
			return file, parent
		}
	}
	return nil, nil
}

func (da *DiskApi) getFileRecursive(id int64, folder *Folder) (*File, *Folder) {
	for _, f := range folder.Files {
		if f.Id == id {
			return f, folder
		}
	}
	for _, f := range folder.Subfolders {
		if file, parent := da.getFileRecursive(id, f); file != nil {
			return file, parent
		}
	}
	return nil, nil
}

func (da *DiskApi) UpdateInfo(
	ctx context.Context,
	id int64,
	newName string,
	isPrivate bool,
) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "diskApi.updateInfo")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	file, _, err := da.Get(ctx, id)
	if err != nil {
		return err
	}

	da.mutex.Lock()
	defer da.mutex.Unlock()

	file.Name = newName
	file.IsPrivate = isPrivate

	// save folder structure to disk
	if err := saveRootFolder(ctx, da.rootPath, da.root); err != nil {
		return fmt.Errorf("file updated, but failed to save structure info: %w", err)
	}

	log.Debugf("disk api: file [%d] updated", file.Id)

	return nil
}

type SaveFileParams struct {
	Filename  string
	FolderId  int64
	Size      int64
	FileType  string
	File      io.Reader
	IsPrivate bool
}

func (da *DiskApi) Save(ctx context.Context, params SaveFileParams) (_ int64, err error) {
	_, span := tracing.GlobalTracer.Start(ctx, "diskApi.save")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	span.SetAttributes(attribute.String("file.name", params.Filename))
	span.SetAttributes(attribute.Int64("file.size", params.Size))
	log.Debugf("disk api: saving new file: %s, folder id: %d", params.Filename, params.FolderId)

	// 1. Initial check with Read Lock
	da.mutex.RLock()
	folder := da.getFolder(da.root, params.FolderId)
	if folder == nil {
		da.mutex.RUnlock()
		return -1, ErrFolderNotFound
	}
	folderPath := folder.Path
	da.mutex.RUnlock()

	log.Debugf("disk api: parent folder found: %s", folderPath)

	// 2. Perform I/O without holding the lock
	newId := NewId()
	newFileName := fmt.Sprintf("%d_%s", newId, params.Filename)
	newFilePath := path.Join(folderPath, newFileName)

	// Check if file already exists (unlikely with timestamp ID, but good practice)
	if _, err := os.Stat(newFilePath); err == nil {
		return -1, fmt.Errorf("file already exists: %s", newFilePath)
	}

	dst, err := os.Create(newFilePath)
	if err != nil {
		return -1, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, params.File); err != nil {
		return -1, err
	}

	// 3. Update structure with Write Lock
	da.mutex.Lock()
	defer da.mutex.Unlock()

	// Re-fetch folder in case it was deleted while we were writing the file
	folder = da.getFolder(da.root, params.FolderId)
	if folder == nil {
		// Cleanup the file we just wrote
		if removeErr := os.Remove(newFilePath); removeErr != nil {
			log.Errorf("failed to remove file after folder not found: %s", removeErr)
		}
		return -1, ErrFolderNotFound
	}

	newFile := &File{
		Id:        newId,
		Name:      params.Filename,
		IsPrivate: params.IsPrivate,
		Path:      newFilePath,
		Type:      params.FileType,
		Size:      params.Size,
		CreatedAt: time.Now(),
	}

	folder.Files[newId] = newFile

	// save folder structure to disk
	if err := saveRootFolder(ctx, da.rootPath, da.root); err != nil {
		return -1, err
	}

	return newId, nil
}

func (da *DiskApi) Get(ctx context.Context, id int64) (*File, *Folder, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	_, span := tracing.GlobalTracer.Start(ctx, "diskApi.getFile")
	defer span.End()

	log.Debugf("disk api: getting file: %d", id)

	file, parent := da.getFile(id)
	if file == nil {
		return nil, nil, ErrFileNotFound
	}

	return file, parent, nil
}

func (da *DiskApi) Delete(ctx context.Context, id int64) error {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	_, span := tracing.GlobalTracer.Start(ctx, "diskApi.delete")
	defer span.End()

	log.Debugf("disk api: deleting file: %d", id)

	file, folder := da.getFile(id)
	if file == nil {
		return ErrFileNotFound
	}

	file, ok := folder.Files[id]
	if !ok {
		return ErrFileNotFound
	}

	if err := os.Remove(file.Path); err != nil {
		return err
	}

	delete(folder.Files, file.Id)

	// save folder structure to disk
	if err := saveRootFolder(ctx, da.rootPath, da.root); err != nil {
		// TODO: send metrics and create alarms for cases like this one
		return fmt.Errorf("file deleted, but failed to save structure info: %w", err)
	}

	log.Debugf("disk api: file [%d] deleted", file.Id)

	return nil
}

func (da *DiskApi) DeleteFolder(ctx context.Context, folderId int64) (err error) {
	_, span := tracing.GlobalTracer.Start(ctx, "diskApi.deleteFolder")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if folderId == 0 {
		return errors.New("cannot delete root folder")
	}

	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: deleting folder: %d", folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return ErrFolderNotFound
	}

	parentFolder := da.getFolder(da.root, folder.ParentId)
	if parentFolder == nil {
		return fmt.Errorf("cannot find parent folder %d", folder.ParentId)
	}

	if err := os.RemoveAll(folder.Path); err != nil {
		return err
	}

	var subfolders []*Folder
	for _, sf := range parentFolder.Subfolders {
		if sf.Id != folder.Id {
			subfolders = append(subfolders, sf)
		}
	}
	parentFolder.Subfolders = subfolders

	// save folder structure to disk
	if err := saveRootFolder(ctx, da.rootPath, da.root); err != nil {
		return fmt.Errorf("folder %d deleted, but failed to save structure info: %w", folderId, err)
	}

	log.Debugf("disk api: folder [%d] [%s] deleted", folderId, folder.Name)

	return nil
}

func (da *DiskApi) GetRootFolder() (*Folder, error) {
	if da.root == nil {
		return nil, errors.New("root folder not created / nil")
	}
	return da.root, nil
}

func (da *DiskApi) GetFolder(ctx context.Context, id int64) (*Folder, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	_, span := tracing.GlobalTracer.Start(ctx, "diskApi.getFolder")
	defer span.End()

	log.Debugf("disk api: getting folder id: %d", id)

	folder := da.getFolder(da.root, id)
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	return folder, nil
}

func (da *DiskApi) NewFolder(ctx context.Context, parentId int64, name string) (*Folder, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	_, span := tracing.GlobalTracer.Start(ctx, "diskApi.newFolder")
	defer span.End()

	log.Debugf("disk api: creating new child folder for: %d", parentId)

	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return nil, errors.New("invalid folder name")
	}

	parentFolder := da.getFolder(da.root, parentId)
	if parentFolder == nil {
		return nil, fmt.Errorf("parent folder [%d]: %w", parentId, ErrFolderNotFound)
	}

	for _, subFolder := range parentFolder.Subfolders {
		if subFolder.Name == name {
			return nil, fmt.Errorf("%s: %w", name, ErrFolderExists)
		}
	}

	newPath := path.Join(parentFolder.Path, name)
	if err := os.Mkdir(newPath, 0755); err != nil {
		return nil, fmt.Errorf("create child folder [%s]: %s", name, err)
	} else {
		log.Debugf("new folder created: %s", name)
	}

	newFolder := &Folder{
		Id:         NewId(),
		ParentId:   parentId,
		Name:       name,
		Path:       newPath,
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
		CreatedAt:  time.Now(),
	}
	parentFolder.Subfolders = append(parentFolder.Subfolders, newFolder)

	// save folder structure to disk
	if err := saveRootFolder(ctx, da.rootPath, da.root); err != nil {
		return nil, fmt.Errorf("child folder created, but failed to save structure info: %w", err)
	}

	return newFolder, nil
}
