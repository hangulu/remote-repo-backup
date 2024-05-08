package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

var numDownloadedRE = regexp.MustCompile(`Transferred:\s+(\d+).*,\s+100%`)
var sizeDownloadedRE = regexp.MustCompile(`Transferred:\s+(\d+.*)\s\/\s.*ETA`)
var numErrorsRE = regexp.MustCompile(`Errors:\s+(\d+)`)

type Metadata struct {
	StartedAt          time.Time
	EndedAt            time.Time
	NumFilesDownloaded int
	SizeDownloaded     string
	NumErrors          int
}

func getDuration(copyAll bool, metadata []Metadata) string {
	if copyAll || (len(metadata) == 0) {
		log.Println("Targeting all files")
		return "off"
	}
	duration := time.Since(metadata[0].StartedAt)
	log.Println("Targeting files since", metadata[0].StartedAt)
	return fmt.Sprintf("%dh", int64(math.Round(duration.Hours())))
}

func ProcessStdoutLine(line string) (int, string, int) {
	files := 0
	size := ""
	errors := 0

	numDownloadedMatches := numDownloadedRE.FindStringSubmatch(line)
	if len(numDownloadedMatches) > 1 {
		convertedFiles, err := strconv.Atoi(numDownloadedMatches[1])
		if err == nil {
			files = convertedFiles
		}
	}
	sizeDownloadedMatches := sizeDownloadedRE.FindStringSubmatch(line)
	if len(sizeDownloadedMatches) > 1 {
		size = sizeDownloadedMatches[1]
	}
	numErrorsMatches := numErrorsRE.FindStringSubmatch(line)
	if len(numErrorsMatches) > 1 {
		convertedErrors, err := strconv.Atoi(numErrorsMatches[1])
		if err == nil {
			errors = convertedErrors
		}
	}
	return files, size, errors
}

func main() {
	copyAll := flag.Bool("copy-all", false, "Copy all files in the remote directory. When false, only the files changed since the last invocation are copied.")
	backupDestination := flag.String("dest", "", "The destination of the backup. Required.")
	backupSource := flag.String("source", "google-drive:", "The remote source of the backup, pre-configured with rclone.")
	flag.Parse()

	if len(*backupDestination) == 0 {
		log.Fatal(errors.New("the backup destination must be specified"))
	}
	metadataFilepath := filepath.Join(*backupDestination, "metadata.json")

	var existingMetadata []Metadata
	rawExistingMetadata, err := os.ReadFile(metadataFilepath)
	if err != nil {
		log.Printf("no existing metadata file. one will be created")
	} else {
		err = json.Unmarshal(rawExistingMetadata, &existingMetadata)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Starting backup")
	if _, err := os.Stat(*backupDestination); err != nil {
		if os.IsNotExist(err) {
			log.Printf("the backup destination was not found at `%s`", *backupDestination)
		}
		log.Fatal(err)
	}

	metadata := Metadata{}
	metadata.StartedAt = time.Now()

	cmd := exec.Command("rclone", "copy", *backupSource, *backupDestination, "--progress", "--max-age", getDuration(*copyAll, existingMetadata))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdoutScanner := bufio.NewScanner(stdout)

	log.Println("Downloading...")
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	numFilesDownloaded := 0
	sizeDownloaded := ""
	numErrors := 0
	for stdoutScanner.Scan() {
		files, size, errors := ProcessStdoutLine(stdoutScanner.Text())
		if files > 0 {
			numFilesDownloaded = files
		}
		if size != "" {
			sizeDownloaded = size
		}
		if errors > 0 {
			numErrors = errors
		}
	}

	if stdoutScanner.Err() != nil {
		cmd.Process.Kill()
		cmd.Wait()
		log.Fatal(stdoutScanner.Err())
	}
	cmd.Wait()
	metadata.EndedAt = time.Now()
	metadata.NumFilesDownloaded = numFilesDownloaded
	metadata.SizeDownloaded = sizeDownloaded
	metadata.NumErrors = numErrors

	updatedMetadata := append([]Metadata{metadata}, existingMetadata...)

	metadataJSON, err := json.Marshal(updatedMetadata)
	if err != nil {
		log.Println("Unable to create JSON from metadata, but the backup is complete")
		log.Fatal(err)
	}

	log.Println("Completed backup")
	log.Println("Recording this backup's metadata: ", metadata)
	f, err := os.OpenFile(metadataFilepath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Unable to record metadata, but the backup is complete")
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s\n", metadataJSON); err != nil {
		log.Fatal(err)
	}

	log.Println("Bye")
}
