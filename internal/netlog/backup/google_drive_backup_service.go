package backup

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"google.golang.org/api/drive/v3"
)

const (
	rootFolderName = "netlog-backup"
)

type GoogleDriveBackupService struct {
	service         *drive.Service
	root            *drive.FileList
	backupsFolderId string
}

func NewGoogleDriveBackupService(httpClient *http.Client) (*GoogleDriveBackupService, error) {
	// TODO:
	// gdService, err := drive.NewService(context.Background(), option.WithCredentialsJSON(credentialsFileBytes))

	driveService, err := drive.New(httpClient)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve drive client: %w", err)
	}

	driveRoot, err := driveService.
		Files.List().
		Fields("files(id, name)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %w", err)
	}

	backupsFolderId := ""
	for _, f := range driveRoot.Files {
		if f.Name == rootFolderName {
			// root backups folder found, get out
			backupsFolderId = f.Id
			break
		}
	}

	s := &GoogleDriveBackupService{
		service: driveService,
		root:    driveRoot,
	}

	if backupsFolderId == "" {
		backupsFolderId, err = s.createRootBackupsFolder()
		if err != nil {
			return nil, fmt.Errorf("failed to create root backups folder: %w", err)
		}
		log.Printf("root backups folder created: %s", backupsFolderId)
	}

	s.backupsFolderId = backupsFolderId

	log.Printf("backups folder ID: %s", s.backupsFolderId)

	return s, nil
}

func (s *GoogleDriveBackupService) DoBackup() error {
	backupFiles, err := s.getNetlogBackupFiles(s.backupsFolderId)
	if err != nil {
		return err
	}

	if len(backupFiles) == 0 {
		log.Println("backups empty, creating initial backup file ...")
		if initialBackupFile, err := s.createInitialBackupFile(); err == nil {
			log.Printf("initial backup created: %s", initialBackupFile.Id)
		} else {
			return err
		}
	} else {
		log.Println("current backup files:")
		for _, i := range backupFiles {
			log.Printf(" -- %s (%s)\n", i.Name, i.Id)
		}
	}

	// TODO:

	return nil
}

func (s *GoogleDriveBackupService) createRootBackupsFolder() (string, error) {
	backupsFolderMeta := &drive.File{
		Name:     rootFolderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	bfRes, err := s.service.
		Files.Create(backupsFolderMeta).
		Fields("id").
		Do()
	if err != nil {
		return "", err
	}

	return bfRes.Id, nil
}

func (s *GoogleDriveBackupService) createInitialBackupFile() (*drive.File, error) {
	initialBackupMeta := &drive.File{
		Name: "initial-backup.json",
		// https://developers.google.com/drive/api/v3/mime-types
		MimeType: "application/vnd.google-apps.file",
		Parents:  []string{s.backupsFolderId},
	}

	testJsonReader := strings.NewReader(`{"va": "test"}`)

	initialBackupFile, err := s.service.
		Files.Create(initialBackupMeta).
		Fields("id, parents").
		Media(testJsonReader).
		Do()
	if err != nil {
		return nil, err
	}

	return initialBackupFile, nil
}

func (s *GoogleDriveBackupService) getNetlogBackupFiles(netlogBackupFolderId string) ([]*drive.File, error) {
	nbQuery := fmt.Sprintf("'%s' in parents and mimeType != 'application/vnd.google-apps.folder' and trashed = false", netlogBackupFolderId)
	backups, err := s.service.
		Files.List().
		Q(nbQuery).
		Fields("files(id, name)").
		Do()
	if err != nil {
		return nil, err
	}

	return backups.Files, nil
}

func (s *GoogleDriveBackupService) ListAllFiles() {
	log.Println("all files:")
	if len(s.root.Files) == 0 {
		log.Println(" -- no files found")
	} else {
		for _, i := range s.root.Files {
			log.Printf(" -- %s (%s)\n", i.Name, i.Id)
		}
	}
}
