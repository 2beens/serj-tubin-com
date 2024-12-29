package main

//// Small CLI tool used to backfill the database with tracks from a Last.fm backup file.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/2beens/serjtubincom/internal/db"
	"github.com/2beens/serjtubincom/internal/spotify"
)

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse and validate the input
	host, port, dbName, jsonPath, verbose, err := parseAndValidateInput()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print the validated inputs for demonstration purposes
	log.Printf("PostgreSQL Host: %s\n", host)
	log.Printf("PostgreSQL Port: %s\n", port)
	log.Printf("PostgreSQL DB Name: %s\n", dbName)
	log.Printf("JSON Path: %s\n", jsonPath)

	repo, err := getRepo(ctx, port, host, dbName)
	if err != nil {
		log.Fatalf("Failed to get repo: %v\n", err)
	}

	// Read the JSON file
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("Failed to read file: %v\n", err)
	}

	// Unmarshal the JSON data into the dataRoot struct
	var lfmRoot dataRoot
	if err := json.Unmarshal(jsonData, &lfmRoot); err != nil {
		log.Fatalf("Failed to parse JSON: %v\n", err)
	}

	// Iterate over tracks, map them to Spotify tracks, and insert them into the database
	var failedInserts []track
	for _, wrapper := range lfmRoot {
		for _, lfmTrack := range wrapper.Track {
			spotifyTrack, err := mapLastFMTrackToSpotifyTrack(lfmTrack)
			if err != nil {
				log.Printf("Failed to map track %+v: %v", lfmTrack, err)
				continue
			}

			// Insert track into DB
			if err = repo.Add(ctx, spotifyTrack); err != nil {
				log.Printf(
					"--- Failed to insert track [%s - %s] [played at: %s]: %v\n",
					spotifyTrack.Artists, spotifyTrack.Name, spotifyTrack.PlayedAt, err,
				)
				failedInserts = append(failedInserts, lfmTrack)
			} else if verbose {
				log.Printf("+++ Inserted track: %+v", spotifyTrack)
			}
		}
	}

	// finally, print the failed inserts as json so we can investigate them separately and fix them
	if len(failedInserts) > 0 {
		failedInsertsJSON, err := json.MarshalIndent(failedInserts, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal failed inserts: %v\n", err)
		}
		log.Println("----------------------------------------------------")
		log.Println("----------------------------------------------------")
		log.Println("")
		log.Printf("Failed inserts below: \n")
		log.Println(string(failedInsertsJSON))
	}
}

func getRepo(ctx context.Context, port string, host string, dbName string) (*spotify.Repo, error) {
	dbPool, err := db.NewDBPool(ctx, db.NewDBPoolParams{
		DBHost:         host,
		DBPort:         port,
		DBName:         dbName,
		TracingEnabled: false,
	})
	if err != nil {
		return nil, fmt.Errorf("new db pool: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return spotify.NewRepo(dbPool), nil
}

func parseAndValidateInput() (string, string, string, string, bool, error) {
	// Define flags for PostgreSQL host, port, database name, and JSON file path
	host := flag.String("host", "", "PostgreSQL host (e.g., localhost or IP address)")
	port := flag.String("port", "", "PostgreSQL port (e.g., 5432)")
	dbName := flag.String("dbname", "", "PostgreSQL database name")
	jsonPath := flag.String("json", "", "Path to the JSON file with LastFM data")
	verbose := flag.Bool("verbose", false, "Verbose output")

	// Parse the flags
	flag.Parse()

	// Validate required inputs
	if *host == "" {
		return "", "", "", "", false, fmt.Errorf("PostgreSQL host is required (use -host)")
	}
	if *port == "" {
		return "", "", "", "", false, fmt.Errorf("PostgreSQL port is required (use -port)")
	}
	if *dbName == "" {
		return "", "", "", "", false, fmt.Errorf("PostgreSQL database name is required (use -dbname)")
	}
	if *jsonPath == "" {
		return "", "", "", "", false, fmt.Errorf("Path to JSON file is required (use -json)")
	}

	// Check if the JSON file exists
	if _, err := os.Stat(*jsonPath); os.IsNotExist(err) {
		return "", "", "", "", false, fmt.Errorf("JSON file does not exist at path: %s", *jsonPath)
	}

	// Return the validated inputs
	return *host, *port, *dbName, *jsonPath, *verbose, nil
}
