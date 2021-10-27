package file_box

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// files can reside in main dir, in e.g. /var
// inside, a subfolders can be created by the client (e.g. photos, pdfs, ...)
// in each subfolder, we have the files named like:
//		<timestamp-nanosecond>_<file-name>.<extension>

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

func (da *DiskApi) getFolder(parent *Folder, id int) *Folder {
	if id == parent.Id {
		return parent
	}
	for _, f := range da.root.Subfolders {
		if sf := da.getFolder(f, id); sf != nil {
			return sf
		}
	}
	return nil
}

func (da *DiskApi) Save(filename string, folderId int, file io.Reader) (int, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("saving new file: %s, folder id: %d", filename, folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return -1, ErrFolderNotFound
	}

	timestampNs := time.Now().Nanosecond()
	newFileName := fmt.Sprintf("%d_%s", timestampNs, filename)
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
		Id:   timestampNs,
		Name: newFileName,
		Path: newFilePath,
	}

	folder.Files[timestampNs] = newFile

	// save folder structure to disk
	if err := saveRootFolder(da.rootPath, da.root); err != nil {
		return -1, err
	}

	return timestampNs, nil
}

func (da *DiskApi) Get(id, folderId int) (*File, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("getting file: %d, folder id: %d", id, folderId)

	folder := da.getFolder(da.root, folderId)
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	file, ok := folder.Files[folderId]
	if !ok {
		return nil, ErrFileNotFound
	}

	return file, nil
}

func (da *DiskApi) GetFolder(id int) (*Folder, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("getting folder id: %d", id)

	folder := da.getFolder(da.root, id)
	if folder == nil {
		return nil, ErrFolderNotFound
	}

	return folder, nil
}

func (da *DiskApi) ListFiles(folderId int) ([]*File, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	log.Debugf("getting files list from folder id: %d", folderId)

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
