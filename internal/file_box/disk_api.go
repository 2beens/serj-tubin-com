package file_box

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	ErrFolderNotFound = errors.New("folder not found")
	ErrFileNotFound   = errors.New("file not found")
)

type DiskApi struct {
	rootPath string
	root     *Folder
	mutex    sync.RWMutex
}

func NewDiskApi(rootPath string) (*DiskApi, error) {
	root, err := getRootFolder(rootPath)
	if err != nil {
		return nil, err
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

func (da *DiskApi) UpdateFileInfo(
	id int64,
	folderId int64,
	newName string,
	isPrivate bool,
) error {
	file, err := da.Get(id, folderId)
	if err != nil {
		return err
	}

	da.mutex.Lock()
	defer da.mutex.Unlock()

	file.Name = newName
	file.IsPrivate = isPrivate

	// save folder structure to disk
	if err := saveRootFolder(da.rootPath, da.root); err != nil {
		return fmt.Errorf("file updated, but failed to save structure info: %w", err)
	}

	log.Debugf("disk api: file [%d] updated", file.Id)

	return nil
}

func (da *DiskApi) Save(
	filename string,
	folderId int64,
	size int64,
	fileType string,
	file io.Reader,
) (int64, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: saving new file: %s, folder id: %d", filename, folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return -1, ErrFolderNotFound
	}

	log.Debugf("disk api: parent folder found: %s", folder.Path)

	newId := NewId()
	newFileName := fmt.Sprintf("%d_%s", newId, filename)
	newFilePath := path.Join(folder.Path, newFileName)
	dst, err := os.Create(newFilePath)
	if err != nil {
		return -1, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return -1, err
	}

	newFile := &File{
		Id:        newId,
		Name:      filename,
		IsPrivate: true,
		Path:      newFilePath,
		Type:      fileType,
		Size:      size,
	}

	folder.Files[newId] = newFile

	// save folder structure to disk
	if err := saveRootFolder(da.rootPath, da.root); err != nil {
		return -1, err
	}

	return newId, nil
}

func (da *DiskApi) Get(id, folderId int64) (*File, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: getting file: %d, folder id: %d", id, folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	file, ok := folder.Files[id]
	if !ok {
		return nil, ErrFileNotFound
	}

	return file, nil
}

func (da *DiskApi) Delete(id, folderId int64) error {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: deleting file: %d, folder id: %d", id, folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return ErrFolderNotFound
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
	if err := saveRootFolder(da.rootPath, da.root); err != nil {
		return fmt.Errorf("file deleted, but failed to save structure info: %w", err)
	}

	log.Debugf("disk api: file [%d] deleted", file.Id)

	return nil
}

func (da *DiskApi) DeleteFolder(folderId int64) error {
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
	if folder == nil {
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
	if err := saveRootFolder(da.rootPath, da.root); err != nil {
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

func (da *DiskApi) GetFolder(id int64) (*Folder, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: getting folder id: %d", id)

	folder := da.getFolder(da.root, id)
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	return folder, nil
}

func (da *DiskApi) NewFolder(parentId int64, name string) (*Folder, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: creating new child folder for: %d", parentId)

	parentFolder := da.getFolder(da.root, parentId)
	if parentFolder == nil {
		return nil, ErrFolderNotFound
	}

	for _, subFolder := range parentFolder.Subfolders {
		if subFolder.Name == name {
			return nil, fmt.Errorf("child folder [%s] already exists", name)
		}
	}

	newPath := path.Join(parentFolder.Path, name)
	if err := os.Mkdir(newPath, 0755); err != nil {
		return nil, fmt.Errorf("create child folder [%s]: %s", name, err)
	} else {
		log.Printf("new folder created: %s", name)
	}

	newFolder := &Folder{
		Id:         NewId(),
		ParentId:   parentId,
		Name:       name,
		Path:       newPath,
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
	}
	parentFolder.Subfolders = append(parentFolder.Subfolders, newFolder)

	// save folder structure to disk
	if err := saveRootFolder(da.rootPath, da.root); err != nil {
		return nil, fmt.Errorf("child folder created, but failed to save structure info: %w", err)
	}

	return newFolder, nil
}

func (da *DiskApi) ListFiles(folderId int64) ([]*File, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("disk api: getting files list from folder id: %d", folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	var files []*File
	for _, f := range folder.Files {
		files = append(files, f)
	}

	return files, nil
}
