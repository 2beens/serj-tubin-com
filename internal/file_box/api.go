package file_box

import "io"

var _ Api = (*DiskApi)(nil)

type Api interface {
	Get(id, folderId int) (*File, error)
	Save(filename string, folderId int, size int64, fileType string, file io.Reader) (int, error)
	Delete(id, folderId int) error
	GetRootFolder() (*Folder, error)
	GetFolder(id int) (*Folder, error)
	NewFolder(parentId int, name string) (*Folder, error)
	ListFiles(folderId int) ([]*File, error)
}
