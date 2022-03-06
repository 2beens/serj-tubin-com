package file_box

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"time"

	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
)

const (
	// json file name for marshaled root folder
	// it is saved within the root folder path
	dirStructureJsonFileName = "root-folder.json"
)

type File struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	IsPrivate bool      `json:"is_private"`
	Path      string    `json:"path"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// FileInfo used for clients, to hide the path
type FileInfo struct {
	Id        int64       `json:"id"`
	ParentId  int64       `json:"parent_id"`
	Name      string      `json:"name"`
	IsPrivate bool        `json:"is_private"`
	IsFile    bool        `json:"is_file"`
	File      string      `json:"file,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	Children  []*FileInfo `json:"children,omitempty"`
}

type Folder struct {
	Id         int64           `json:"id"`
	ParentId   int64           `json:"parent_id"`
	Name       string          `json:"name"`
	Path       string          `json:"path"`
	Subfolders []*Folder       `json:"subfolders"`
	Files      map[int64]*File `json:"files"`
	CreatedAt  time.Time       `json:"created_at"`
}

// NewId returns a simple unix time in micro
// fair enough for use case of a simple folder/file ID
func NewId() int64 {
	return time.Now().UnixMicro()
}

func NewRootFolder(path string) *Folder {
	return &Folder{
		Id:         0,
		Name:       "root",
		Path:       path,
		Subfolders: []*Folder{},
		Files:      make(map[int64]*File),
	}
}

func NewFolderInfo(parentId int64, folder *Folder) *FileInfo {
	folderInfo := &FileInfo{
		Id:        folder.Id,
		ParentId:  parentId,
		Name:      folder.Name,
		Children:  []*FileInfo{},
		IsFile:    false,
		CreatedAt: folder.CreatedAt,
	}

	sort.Slice(folder.Subfolders, func(i, j int) bool {
		return folder.Subfolders[i].Name < folder.Subfolders[j].Name
	})
	for _, subFolder := range folder.Subfolders {
		folderInfo.Children = append(folderInfo.Children, NewFolderInfo(folder.Id, subFolder))
	}

	files := make([]*File, 0, len(folder.Files))
	for _, f := range folder.Files {
		files = append(files, f)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	for _, file := range files {
		folderInfo.Children = append(folderInfo.Children, &FileInfo{
			Id:        file.Id,
			ParentId:  folder.Id,
			IsPrivate: file.IsPrivate,
			Name:      file.Name,
			File:      file.Type,
			IsFile:    true,
			CreatedAt: file.CreatedAt,
		})
	}

	return folderInfo
}

func rootPathExists(rootPath string) error {
	exists, err := pkg.PathExists(rootPath, true)
	if err != nil {
		return fmt.Errorf("check root path %s: %s", rootPath, err)
	}
	if !exists {
		return fmt.Errorf("root path [%s] does not exist", rootPath)
	}
	return nil
}

func getRootFolder(rootPath string) (*Folder, error) {
	if err := rootPathExists(rootPath); err != nil {
		return nil, err
	}

	folderStructureJsonPath := path.Join(rootPath, dirStructureJsonFileName)
	log.Debugf("loading folder structure from: %s", folderStructureJsonPath)

	rootFolderJsonExists, err := pkg.PathExists(folderStructureJsonPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to check existance of root folder [%s]: %s", folderStructureJsonPath, err)
	}

	if !rootFolderJsonExists {
		log.Debugln("root folder JSON does not exist, creating a fresh copy ...")
		rootFolder := NewRootFolder(rootPath)
		if err := saveRootFolder(rootPath, rootFolder); err != nil {
			return nil, fmt.Errorf("root folder created, but failed to save: %w", err)
		}
		return rootFolder, nil
	}

	rootFolderJson, err := os.ReadFile(folderStructureJsonPath)
	if err != nil {
		return nil, err
	}
	var rootFolder Folder
	if err := json.Unmarshal(rootFolderJson, &rootFolder); err != nil {
		return nil, fmt.Errorf("failed to unmarshal root folder: %w", err)
	}
	return &rootFolder, nil
}

func saveRootFolder(rootPath string, folder *Folder) error {
	if err := rootPathExists(rootPath); err != nil {
		return err
	}

	folderStructureJsonPath := path.Join(rootPath, dirStructureJsonFileName)
	log.Debugf("saving folder structure to: %s", folderStructureJsonPath)

	rootFolderJson, err := json.Marshal(folder)
	if err != nil {
		return err
	}

	if err := os.Remove(folderStructureJsonPath); err != nil {
		log.Warnf("remove current root folder json: %s", err)
	}

	dst, err := os.Create(folderStructureJsonPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, bytes.NewReader(rootFolderJson)); err != nil {
		return err
	}

	log.Debugln("new folder structure saved")

	return nil
}
