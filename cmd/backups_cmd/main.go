package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"context"
	"encoding/json"
	"net/http"
	"os"

	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	rootFolderName = "netlog-backup"
)

func main() {
	credentialsFile := flag.String(
		"gd-creds",
		"./lazar-dusan-veliki-drive-credentials.json",
		"lazar dusan google drive credentials json",
	)
	logsPath := flag.String("logs-path", "", "server logs file path (empty for stdout)")

	flag.Parse()

	loggingSetup(*logsPath)

	log.Println("staring netlog backup ...")

	if *credentialsFile == "" {
		log.Fatalln("google drive credentials json not specified")
	}

	// lazar.dusan.veliki@gmail.com // stara sifra
	credentialsFileBytes, err := ioutil.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credentialsFileBytes, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	gdService, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	root, err := gdService.
		Files.List().
		PageSize(100).
		Fields("nextPageToken, files(id, name)").
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	log.Println("existing files:")
	if len(root.Files) == 0 {
		log.Println(" -- no files found")
	} else {
		for _, i := range root.Files {
			log.Printf(" -- %s (%s)\n", i.Name, i.Id)
		}
	}

	if err := checkRootBackupsFolder(gdService, root); err != nil {
		log.Fatalf("failed to check root backups folder: %s", err)
	}

	// TODO: do backup
}

func checkRootBackupsFolder(gdService *drive.Service, root *drive.FileList) error {
	for _, i := range root.Files {
		if i.Name == rootFolderName {
			// root backups folder found, get out
			return nil
		}
	}

	backupsFolderMeta := &drive.File{
		Name:     rootFolderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	bfRes, err := gdService.
		Files.Create(backupsFolderMeta).
		Fields("id").
		Do()
	if err != nil {
		return err
	}

	log.Printf("root backups folder created: %s", bfRes.Id)

	return nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func loggingSetup(logFileName string) {
	if logFileName == "" {
		log.SetOutput(os.Stdout)
		return
	}

	if !strings.HasSuffix(logFileName, ".log") {
		logFileName += ".log"
	}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Panicf("failed to open log file %q: %s", logFileName, err)
	}

	log.SetOutput(logFile)
}
