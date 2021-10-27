package file_box

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
)

// files can reside in main dir, in e.g. /var
// inside, a subfolders can be created by the client (e.g. photos, pdfs, ...)
// in each subfolder, we have the files named like:
//		<uuid>_<file-name>.<extension>

const (
	// json file name for marshaled root folder
	// it is saved within the root folder path
	dirStructureJsonFileName = "root-folder.json"
)

type DiskApi struct {
	root *Folder
	// foldersMap map[int]*Folder
}

func NewDiskApi(rootPath string) (*DiskApi, error) {
	exists, err := pkg.PathExists(rootPath, true)
	if err != nil {
		return nil, fmt.Errorf("check path %s: %s", rootPath, err)
	}
	if !exists {
		return nil, fmt.Errorf("path [%s] does not exist", rootPath)
	}

	folderStructureJsonPath := path.Join(rootPath, dirStructureJsonFileName)
	log.Debugf("loading folder structure from: %s", folderStructureJsonPath)

	var rootDir *Folder = nil
	rootDirJsonExists, err := pkg.PathExists(folderStructureJsonPath, false)
	if err != nil {
		log.Warnf("failed to check existance of root folder [%s]: %s", folderStructureJsonPath, err)
	} else if rootDirJsonExists {
		rootFolderJson, err := os.ReadFile(folderStructureJsonPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rootFolderJson, rootDir); err != nil {
			return nil, fmt.Errorf("failed to unmarshal root folder: %w", err)
		}
	}

	if rootDir == nil {
		log.Debugf(("creating a fresh root folder ..."))
		rootDir = NewRootFolder(folderStructureJsonPath)
	}

	return &DiskApi{
		root: rootDir,
	}, nil
}

func (da *DiskApi) Save(filename string, dirId int, file io.Reader) error {
	dst, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return err
	}

	return nil
}

func (da *DiskApi) Get(id, dirId int) {
	panic("not implemented")
}

func (da *DiskApi) List(dirId int) []string {
	panic("not implemented")
}
