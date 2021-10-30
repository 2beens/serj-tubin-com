package file_box

import "io"

var _ Api = (*DiskApi)(nil)

type Api interface {
	Get(id, folderId int64) (*File, error)
	Save(filename string, folderId int64, size int64, fileType string, file io.Reader) (int64, error)
	Delete(id, folderId int64) error
	GetRootFolder() (*Folder, error)
	GetFolder(id int64) (*Folder, error)
	DeleteFolder(folderId int64) error
	NewFolder(parentId int64, name string) (*Folder, error)
	ListFiles(folderId int64) ([]*File, error)
}
