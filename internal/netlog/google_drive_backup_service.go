package netlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	rootBackupsFolderName = "netlog-backup"
	visitsFileChunkSize   = 350 // number of visits in one backup file
)

type GoogleDriveBackupService struct {
	psqlApi         *PsqlApi
	service         *drive.Service
	backupsFolderId string
}

func NewGoogleDriveBackupService(credentialsJson []byte) (*GoogleDriveBackupService, error) {
	// https://github.com/googleapis/google-api-go-client/blob/master/drive/v3/drive-gen.go
	ctx := context.Background()
	driveService, err := drive.NewService(ctx, option.WithCredentialsJSON(credentialsJson))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve drive client: %w", err)
	}

	rootFolderQuery := fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and trashed = false and name = '%s'", rootBackupsFolderName)
	log.Println(rootFolderQuery)
	netlogBackupFolder, err := driveService.
		Files.List().
		Q(rootFolderQuery).
		Fields("files(id, name)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %w", err)
	}

	backupsFolderId := ""
	if len(netlogBackupFolder.Files) == 1 {
		rbf := netlogBackupFolder.Files[0]
		log.Printf("root backups folder found, %s: %s", rbf.Name, rbf.Id)
		backupsFolderId = rbf.Id
	} else if len(netlogBackupFolder.Files) == 0 {
		log.Println("root backups folder not found, will recreate")
	} else {
		rbf := netlogBackupFolder.Files[0]
		log.Printf("attention: found %d root backups folders, will take the first one: %s", len(netlogBackupFolder.Files), rbf.Id)
		backupsFolderId = rbf.Id
	}

	psqlApi, err := NewNetlogPsqlApi()
	if err != nil {
		return nil, fmt.Errorf("failed to create PSQL api client: %w", err)
	}

	s := &GoogleDriveBackupService{
		psqlApi: psqlApi,
		service: driveService,
	}

	if backupsFolderId == "" {
		log.Println("root backups folder not found, recreating ...")
		backupsFolderId, err = s.createRootBackupsFolder()
		if err != nil {
			return nil, fmt.Errorf("failed to create root backups folder: %w", err)
		}
		log.Printf("new root backups folder created: %s", backupsFolderId)
	} else {
		log.Printf("found backups folder ID: %s", backupsFolderId)
	}

	s.backupsFolderId = backupsFolderId

	return s, nil
}

func (s *GoogleDriveBackupService) Reinit(baseTime time.Time) error {
	log.Println("netlog visits backup reinit starting ...")

	err := s.service.Files.
		Delete(s.backupsFolderId).
		Do()
	if err != nil {
		return err
	}

	backupsFolderId, err := s.createRootBackupsFolder()
	if err != nil {
		return fmt.Errorf("failed to create root backups folder: %w", err)
	}

	log.Printf("new root backups folder created: %s", backupsFolderId)

	s.backupsFolderId = backupsFolderId

	return s.DoBackup(baseTime)
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
		Name:     rootBackupsFolderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	bfRes, err := s.service.
		Files.Create(backupsFolderMeta).
		Fields("id").
		Do()
	if err != nil {
		return "", err
	}

	if pId, err := s.updateFilePermission(bfRes.Id); err != nil {
		return bfRes.Id, fmt.Errorf("failed to create additional permission for root backup folder: %s", err)
	} else {
		log.Printf("permission %s created for root backup folder %s", pId, bfRes.Id)
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

	if len(visits) < visitsFileChunkSize {
		toIndex = len(visits)
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
			return fmt.Errorf("%s: failed to create visits backups file: %w", nextFileName, err)
		}

		permissionId, err := s.updateFilePermission(nextBackupChunkFile.Id)
		if err != nil {
			return fmt.Errorf("%s: failed to create additional permission: %s", nextFileName, err)
		}

		log.Printf("%s: backup file [%s] [permission %s] saved: %s", nextFileName, fileMeta.Name, permissionId, nextBackupChunkFile.Id)

		fromIndex = toIndex
		toIndex = toIndex + visitsFileChunkSize
		if toIndex >= len(visits) {
			toIndex = len(visits)
		}
	}

	return nil
}

func (s *GoogleDriveBackupService) updateFilePermission(fileId string) (string, error) {
	permission := &drive.Permission{
		EmailAddress: "lazar.dusan.veliki@gmail.com",
		Type:         "user",
		Role:         "reader",
	}

	createdPermission, err := s.service.Permissions.
		Create(fileId, permission).
		Do()
	if err != nil {
		return "", err
	}

	return createdPermission.Id, nil
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
