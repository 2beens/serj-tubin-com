package file_box

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
)

const (
	// json file name for marshaled root folder
	// it is saved within the root folder path
	dirStructureJsonFileName = "root-folder.json"
)

type File struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	// Size int    `json:"size"`
}

type Folder struct {
	Id         int           `json:"id"`
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Subfolders []*Folder     `json:"subfolders"`
	Files      map[int]*File `json:"files"`
}

func NewRootFolder(path string) *Folder {
	return &Folder{
		Id:         0,
		Name:       "root",
		Path:       path,
		Subfolders: []*Folder{},
		Files:      make(map[int]*File),
	}
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
		log.Debugf(("root folder JSON does not exist, creating a fresh copy ..."))
		rootFolder := NewRootFolder(folderStructureJsonPath)
		saveRootFolder(rootPath, rootFolder)
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
