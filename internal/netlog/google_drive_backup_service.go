package netlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	rootBackupsFolderName = "netlog-backup"
	visitsFileChunkSize   = 350 // number of visits in one backup file
)

var (
	ErrPermissionNotFound = errors.New("lazar permissions not found")
)

type GoogleDriveBackupService struct {
	psqlApi              *PsqlApi
	service              *drive.Service
	backupsFolderId      string
	lazarDusanPermission *drive.Permission
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

		var permission *drive.Permission
		backupsFolderId, permission, err = s.createRootBackupsFolder()
		if err != nil {
			return nil, fmt.Errorf("failed to create root backups folder: %w", err)
		}

		s.lazarDusanPermission = permission

		log.Printf("new root backups folder created: %s, permission: %s", backupsFolderId, permission.Id)
	} else {
		pList, err := driveService.Permissions.List(backupsFolderId).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get root backup folder permission list: %w", err)
		}

		for _, p := range pList.Permissions {
			//log.Printf(" ====> %+v", p)
			if p.Type == "user" && p.Role == "reader" {
				s.lazarDusanPermission = p
				break
			}
		}

		log.Printf("found backups folder ID: %s", backupsFolderId)
	}

	s.backupsFolderId = backupsFolderId

	if s.lazarDusanPermission == nil {
		return s, ErrPermissionNotFound
	}

	return s, nil
}

func DestroyAllFiles(credentialsJson []byte) error {
	ctx := context.Background()
	driveService, err := drive.NewService(ctx, option.WithCredentialsJSON(credentialsJson))
	if err != nil {
		return fmt.Errorf("unable to retrieve drive client: %w", err)
	}

	log.Println(" !! destroying netlog visits backups ...")

	files, err := driveService.Files.List().Do()
	if err != nil {
		return fmt.Errorf("failed to get files list: %w", err)
	}

	// TODO: in case of more than 100 files, the rest will not be deleted

	for _, f := range files.Files {
		log.Printf("deleting: %s [%s] ...", f.Name, f.Id)
		err = driveService.Files.
			Delete(f.Id).
			Do()
		if err != nil {
			log.Printf("failed to delete file %s: %s", f.Id, err)
		}
	}

	if err := driveService.Files.EmptyTrash().Do(); err != nil {
		log.Printf("empty trash err: %s", err)
	}

	return nil
}

func (s *GoogleDriveBackupService) Reinit(baseTime time.Time) error {
	log.Println("netlog visits backup reinit starting ...")

	err := s.service.Files.
		Delete(s.backupsFolderId).
		Do()
	if err != nil {
		return err
	}

	backupsFolderId, permission, err := s.createRootBackupsFolder()
	if err != nil {
		return fmt.Errorf("failed to create root backups folder: %w", err)
	}

	log.Printf("new root backups folder created: %s, permission: %s", backupsFolderId, permission.Id)

	s.backupsFolderId = backupsFolderId
	s.lazarDusanPermission = permission

	return s.DoBackup(baseTime)
}

func (s *GoogleDriveBackupService) DoBackup(baseTime time.Time) error {
	log.Println("DoBackup start ...")
	if s == nil {
		panic("service is nil")
	}

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

func (s *GoogleDriveBackupService) createRootBackupsFolder() (folderId string, createdPermission *drive.Permission, err error) {
	backupsFolderMeta := &drive.File{
		Name:     rootBackupsFolderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	bfRes, err := s.service.
		Files.Create(backupsFolderMeta).
		Fields("id").
		Do()
	if err != nil {
		return "", nil, err
	}

	log.Printf("creating permission for backup folder [%s] ...", bfRes.Id)

	permission := &drive.Permission{
		EmailAddress: "lazar.dusan.veliki@gmail.com",
		Type:         "user",
		Role:         "reader",
	}

	cp, err := s.service.Permissions.
		Create(bfRes.Id, permission).
		Do()
	if err != nil {
		return "", nil, fmt.Errorf("failed to create permission: %w", err)
	}

	log.Println("permission created")

	return bfRes.Id, cp, nil
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

		log.Printf(
			"%s: create initial backup file with %d netlog visits [from %d to %d] [chunk %d / %d] ...",
			nextFileName, len(nextVisits), fromIndex, toIndex, i, chunks,
		)

		nextVisitsJson, err := json.Marshal(nextVisits)
		if err != nil {
			return fmt.Errorf("%s failed to marshal netlog visits: %w", nextFileName, err)
		}

		log.Printf("%s: creating file on google drive ...", nextFileName)
		fileMeta := &drive.File{
			Name: nextFileName,
			// https://developers.google.com/drive/api/v3/mime-types
			MimeType:      "application/vnd.google-apps.file",
			Parents:       []string{s.backupsFolderId},
			PermissionIds: []string{s.lazarDusanPermission.Id},
			//Permissions:   []*drive.Permission{s.lazarDusanPermission},
		}

		retries := 0

		// goto considered harmful :)
	loop:
		retries++
		nextBackupChunkFile, err := s.service.
			Files.Create(fileMeta).
			Fields("id, parents").
			Media(bytes.NewReader(nextVisitsJson)).
			Do()
		if err != nil {
			if strings.Contains(err.Error(), "internalError") {
				if retries >= 5 {
					return fmt.Errorf("%s: failed to create visits backups file after %d retries: %w", nextFileName, retries, err)
				}
				log.Printf("%s: backup failed, will try again in 10 seconds: %s", nextFileName, err)
				time.Sleep(10 * time.Second)
				goto loop
			}
			return fmt.Errorf("%s: failed to create visits backups file: %w", nextFileName, err)
		}

		log.Printf("%s: backup file [%s] [permission %s] saved: %s", nextFileName, fileMeta.Name, s.lazarDusanPermission.Id, nextBackupChunkFile.Id)

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
