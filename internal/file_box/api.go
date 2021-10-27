package file_box

import "io"

type Api interface {
	Save(filename string, dirId int, file io.Reader) error
	Get(id, dirId int)
	List(dirId int) []string
}

type File struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Size int    `json:"size"`
}

type Folder struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Subfolders []*Folder `json:"subfolders"`
	Files      []*File   `json:"files"`
}

func NewRootFolder(path string) *Folder {
	return &Folder{
		Id:         0,
		Name:       "root",
		Path:       path,
		Subfolders: []*Folder{},
		Files:      []*File{},
	}
}
