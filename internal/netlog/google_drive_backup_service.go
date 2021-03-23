package netlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	rootFolderName      = "netlog-backup"
	visitsFileChunkSize = 350 // number of visits in one backup file
)

type GoogleDriveBackupService struct {
	psqlApi         *PsqlApi
	service         *drive.Service
	root            *drive.FileList
	backupsFolderId string
}

func NewGoogleDriveBackupService(token *oauth2.Token, config *oauth2.Config) (*GoogleDriveBackupService, error) {
	// https://github.com/googleapis/google-api-go-client/blob/master/drive/v3/drive-gen.go
	ctx := context.Background()
	driveService, err := drive.NewService(
		ctx,
		option.WithTokenSource(config.TokenSource(ctx, token)),
	)
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

	log.Printf("all files count: %d", len(driveRoot.Files))

	backupsFolderId := ""
	for _, f := range driveRoot.Files {
		if f.Name == rootFolderName {
			// root backups folder found, get out
			backupsFolderId = f.Id
			log.Printf("root backups folder found, %s: %s", f.Name, f.Id)
			break
		}
	}

	psqlApi, err := NewNetlogPsqlApi()
	if err != nil {
		return nil, fmt.Errorf("failed to create PSQL api client: %w", err)
	}

	s := &GoogleDriveBackupService{
		psqlApi: psqlApi,
		service: driveService,
		root:    driveRoot,
	}

	if backupsFolderId == "" {
		log.Println("root backups folder not found, recreating ...")
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

func (s *GoogleDriveBackupService) DoBackup(baseTime time.Time) error {
	currentAllBackupFiles, err := s.getNetlogBackupFiles(s.backupsFolderId)
	if err != nil {
		return err
	}

	if len(currentAllBackupFiles) == 0 {
		log.Println("backups empty, creating initial backup file ...")
		if err := s.createInitialBackupFile(baseTime); err != nil {
			return err
		}
		log.Println("initial backup files created!")
		return nil
	}

	log.Println("current backup files:")
	lastCreatedAt := time.Time{}
	for _, file := range currentAllBackupFiles {
		createdAt, err := time.Parse(time.RFC3339, file.CreatedTime)
		if err != nil {
			log.Printf(" ---> error parsing created at for file %s: %s", file.Name, err)
			continue
		}
		log.Printf(" -- [%v]: %s (%s)\n", createdAt, file.Name, file.Id)

		if createdAt.After(lastCreatedAt) {
			lastCreatedAt = createdAt
		}
	}

	visitsToBackup, err := s.psqlApi.GetAllVisits(&lastCreatedAt)
	if err != nil {
		return fmt.Errorf("failed to get next backup visits: %w", err)
	}

	if len(visitsToBackup) == 0 {
		log.Println("no new netlog visits to backup, done")
		return nil
	}

	log.Printf(" ---- backing up %d netlog visits since %v", len(visitsToBackup), lastCreatedAt)

	nextBackupFileName := fmt.Sprintf("netlog-visits-%d-%d-%d", baseTime.Day(), baseTime.Month(), baseTime.Year())
	fileCounter := 1
	for {
		nameExists := false
		for _, file := range currentAllBackupFiles {
			if file.Name == (nextBackupFileName + ".json") {
				nameExists = true
				break
			}
		}
		if nameExists {
			fileCounter++
			nextBackupFileName = fmt.Sprintf("%s_%d", nextBackupFileName, fileCounter)
		} else {
			break
		}
	}

	if err := s.backupVisits(visitsToBackup, nextBackupFileName); err != nil {
		return fmt.Errorf("failed to backup visits: %w", err)
	}

	log.Printf("next backup since %v successfully saved: %s", lastCreatedAt, nextBackupFileName)

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

func (s *GoogleDriveBackupService) createInitialBackupFile(baseTime time.Time) error {
	visits, err := s.psqlApi.GetAllVisits(nil)
	if err != nil {
		return fmt.Errorf("failed to get netlog visits from db: %w", err)
	}

	log.Printf("initial backup of %d visits starting ...", len(visits))

	baseFileName := fmt.Sprintf("initial-%d-%d-%d", baseTime.Day(), baseTime.Month(), baseTime.Year())
	if err := s.backupVisits(visits, baseFileName); err != nil {
		return fmt.Errorf("failed to backup visits: %w", err)
	}

	return nil
}

func (s *GoogleDriveBackupService) backupVisits(visits []*Visit, baseFileName string) error {
	chunks := len(visits) / visitsFileChunkSize
	fromIndex, toIndex := 0, visitsFileChunkSize
	if len(visits)%visitsFileChunkSize > 0 {
		chunks++
	}

	// TODO: run in a few goroutines to make faster (if needed)
	for i := 1; i <= chunks; i++ {
		nextFileName := fmt.Sprintf("%s_%d.json", baseFileName, i)
		nextVisits := visits[fromIndex:toIndex]

		log.Printf("%s: create initial backup file with %d netlog visits [from %d to %d] ...", nextFileName, len(nextVisits), fromIndex, toIndex)

		nextVisitsJson, err := json.Marshal(nextVisits)
		if err != nil {
			return fmt.Errorf("%s failed to marshal netlog visits: %w", nextFileName, err)
		}

		log.Printf("%s: creating file on google drive ...", nextFileName)
		fileMeta := &drive.File{
			Name: nextFileName,
			// https://developers.google.com/drive/api/v3/mime-types
			MimeType: "application/vnd.google-apps.file",
			Parents:  []string{s.backupsFolderId},
		}

		nextBackupChunkFile, err := s.service.
			Files.Create(fileMeta).
			Fields("id, parents").
			Media(bytes.NewReader(nextVisitsJson)).
			Do()
		if err != nil {
			return fmt.Errorf("%s: failed to create visits backup file: %w", nextFileName, err)
		}

		log.Printf("%s: backup file [%s] saved: %s", nextFileName, fileMeta.Name, nextBackupChunkFile.Id)

		fromIndex = toIndex
		toIndex = toIndex + visitsFileChunkSize
		if toIndex >= len(visits) {
			toIndex = len(visits)
		}
	}

	return nil
}

func (s *GoogleDriveBackupService) getNetlogBackupFiles(netlogBackupFolderId string) ([]*drive.File, error) {
	nbQuery := fmt.Sprintf("'%s' in parents and mimeType != 'application/vnd.google-apps.folder' and trashed = false", netlogBackupFolderId)
	backups, err := s.service.
		Files.List().
		Q(nbQuery).
		Fields("files(id, name, createdTime)").
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
