package file_box

import (
	"io"
	"sync"
)

type DiskTestApi struct {
	mutex sync.RWMutex
}

func NewDiskTestApi() *DiskTestApi {
	return &DiskTestApi{}
}

func (da *DiskTestApi) Get(id, folderId int64) (*File, error) {
	panic("implement me")
}

func (da *DiskTestApi) UpdateInfo(id int64, folderId int64, newName string, isPrivate bool) error {
	panic("implement me")
}

func (da *DiskTestApi) Save(filename string, folderId int64, size int64, fileType string, file io.Reader) (int64, error) {
	panic("implement me")
}

func (da *DiskTestApi) Delete(id, folderId int64) error {
	panic("implement me")
}

func (da *DiskTestApi) GetRootFolder() (*Folder, error) {
	panic("implement me")
}

func (da *DiskTestApi) GetFolder(id int64) (*Folder, error) {
	panic("implement me")
}

func (da *DiskTestApi) DeleteFolder(folderId int64) error {
	panic("implement me")
}

func (da *DiskTestApi) NewFolder(parentId int64, name string) (*Folder, error) {
	panic("implement me")
}
