package file_box

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFolderInfo(t *testing.T) {
	rootFolder := getTestRootFolder()
	folderInfo := NewFolderInfo(rootFolder)
	assert.NotNil(t, folderInfo)

	folderInfoJson, err := json.Marshal(folderInfo)
	require.NoError(t, err)
	assert.Equal(t,
		`{"id":0,"name":"root","children":[{"id":1,"name":"f1","children":[{"id":11,"name":"f11","children":[{"id":100,"name":"file1"}]}]},{"id":2,"name":"f2"},{"id":100,"name":"file1"}]}`,
		string(folderInfoJson),
	)
}

func getTestRootFolder() *Folder {
	rootFolder := &Folder{
		Id:         0,
		Name:       "root",
		Path:       "/mnt/root",
		Subfolders: []*Folder{},
		Files:      make(map[int]*File),
	}

	f1 := &Folder{
		Id:         1,
		Name:       "f1",
		Path:       "/mnt/f1",
		Subfolders: []*Folder{},
		Files:      make(map[int]*File),
	}
	f2 := &Folder{
		Id:         2,
		Name:       "f2",
		Path:       "/mnt/f2",
		Subfolders: []*Folder{},
		Files:      make(map[int]*File),
	}
	rootFolder.Subfolders = append(rootFolder.Subfolders, f1)
	rootFolder.Subfolders = append(rootFolder.Subfolders, f2)

	file1 := &File{
		Id:   100,
		Name: "file1",
		Path: "/mnt/file1",
	}
	rootFolder.Files[file1.Id] = file1

	f11 := &Folder{
		Id:         11,
		Name:       "f11",
		Path:       "/mnt/f1/f11",
		Subfolders: []*Folder{},
		Files:      make(map[int]*File),
	}
	f1.Subfolders = append(f1.Subfolders, f11)

	file11 := &File{
		Id:   1100,
		Name: "file11",
		Path: "/mnt/f1/file11",
	}
	f11.Files[file11.Id] = file1

	return rootFolder
}
