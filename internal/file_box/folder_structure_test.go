package file_box

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFolderInfo(t *testing.T) {
	rootFolder := getTestRootFolder()
	folderInfo := NewFolderInfo(rootFolder.Id, rootFolder)
	assert.NotNil(t, folderInfo)

	folderInfoJson, err := json.Marshal(folderInfo)
	require.NoError(t, err)
	assert.Equal(t,
		`{"id":0,"parent_id":0,"name":"root","is_private":false,"is_file":false,"created_at":"0001-01-01T00:00:00Z","children":[{"id":1,"parent_id":0,"name":"f1","is_private":false,"is_file":false,"created_at":"2022-01-02T10:10:10Z","children":[{"id":11,"parent_id":1,"name":"f11","is_private":false,"is_file":false,"created_at":"2022-01-11T22:10:10Z","children":[{"id":100,"parent_id":11,"name":"file1","is_private":false,"is_file":true,"created_at":"2022-01-05T10:10:10Z"}]}]},{"id":2,"parent_id":0,"name":"f2","is_private":false,"is_file":false,"created_at":"2022-01-03T10:10:10Z"},{"id":100,"parent_id":0,"name":"file1","is_private":false,"is_file":true,"created_at":"2022-01-05T10:10:10Z"}]}`,
		string(folderInfoJson),
	)
}

func TestNewId(t *testing.T) {
	id := NewId()
	assert.True(t, id > 0)
}

func getTestRootFolder() *Folder {
	rootFolder := &Folder{
		Id:         0,
		Name:       "root",
		Path:       "/mnt/root",
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
	}

	f1 := &Folder{
		Id:         1,
		Name:       "f1",
		Path:       "/mnt/f1",
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
		CreatedAt:  time.Date(2022, 01, 02, 10, 10, 10, 0, time.UTC),
	}
	f2 := &Folder{
		Id:         2,
		Name:       "f2",
		Path:       "/mnt/f2",
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
		CreatedAt:  time.Date(2022, 01, 03, 10, 10, 10, 0, time.UTC),
	}
	rootFolder.Subfolders = append(rootFolder.Subfolders, f1)
	rootFolder.Subfolders = append(rootFolder.Subfolders, f2)

	file1 := &File{
		Id:        100,
		Name:      "file1",
		Path:      "/mnt/file1",
		CreatedAt: time.Date(2022, 01, 05, 10, 10, 10, 0, time.UTC),
	}
	rootFolder.Files[file1.Id] = file1

	f11 := &Folder{
		Id:         11,
		Name:       "f11",
		Path:       "/mnt/f1/f11",
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
		CreatedAt:  time.Date(2022, 01, 11, 22, 10, 10, 0, time.UTC),
	}
	f1.Subfolders = append(f1.Subfolders, f11)

	file11 := &File{
		Id:        1100,
		Name:      "file11",
		Path:      "/mnt/f1/file11",
		CreatedAt: time.Date(2022, 02, 02, 10, 10, 10, 0, time.UTC),
	}
	f11.Files[file11.Id] = file1

	return rootFolder
}
