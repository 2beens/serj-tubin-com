package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/netlog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

func main() {
	credentialsFile := flag.String(
		"gd-creds",
		"./lazar-dusan-veliki-drive-credentials.json",
		"lazar dusan google drive credentials json",
	)
	tokenFile := flag.String(
		"token-file",
		"./token.json",
		"google drive token file json",
	)
	logsPath := flag.String("logs-path", "", "server logs file path (empty for stdout)")

	flag.Parse()

	loggingSetup(*logsPath)

	log.Println("staring netlog backup ...")

	if *credentialsFile == "" {
		log.Fatalln("google drive credentials json not specified")
	}
	if *tokenFile == "" {
		log.Fatalln("google drive token file json not specified")
	}

	// lazar.dusan.veliki@gmail.com // stara sifra
	credentialsFileBytes, err := ioutil.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credentialsFileBytes, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("unable to parse client secret file to config: %v", err)
	}

	token, err := getOauth2Token(*tokenFile, config)
	if err != nil {
		log.Fatalf("failed to get http client: %s", err)
	}

	s, err := netlog.NewGoogleDriveBackupService(token, config)
	if err != nil {
		log.Fatalf("failed to create google drive backup service: %s", err)
	}

	baseTime := time.Now()
	if err := s.DoBackup(baseTime); err != nil {
		log.Fatalf("%+v", err)
	}
}

// Retrieve a token, saves the token, then returns it.
func getOauth2Token(tokenFilePath string, config *oauth2.Config) (*oauth2.Token, error) {
	// the file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first time
	token, err := tokenFromFile(tokenFilePath)
	if err != nil {
		log.Println("failed to get oauth2 token from file, getting from web ...")
		token = getTokenFromWeb(config)
		// save token
		if err := saveToken(tokenFilePath, token); err != nil {
			return nil, fmt.Errorf("failed to save token json: %w", err)
		}
	}
	return token, nil
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
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
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