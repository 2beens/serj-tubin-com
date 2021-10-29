package file_box

import "io"

var _ Api = (*DiskApi)(nil)

type Api interface {
	Save(filename string, folderId int, file io.Reader) (int, error)
	Get(id, folderId int) (*File, error)
	GetRootFolder() (*Folder, error)
	GetFolder(id int) (*Folder, error)
	ListFiles(folderId int) ([]*File, error)
}
