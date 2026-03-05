package history

import (
	"encoding/json"
	"fmt"
	"nmod-cleaner/internal/analyzer"
	"os"
	"path/filepath"
	"time"
)

const historyFile = ".nmod-cleaner-history.json"

// Record represents a single NmodCleaner run.
type Record struct {
	Timestamp  time.Time `json:"timestamp"`
	DirsClean  int       `json:"dirsClean"`
	DirsFailed int       `json:"dirsFailed"`
	BytesFreed int64     `json:"bytesFreed"`
}

// historyPath returns the path to the history file in the user's home directory.
func historyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, historyFile), nil
}

// Append persists a new run record to the history file.
func Append(record Record) error {
	records, err := Load()
	if err != nil {
		records = []Record{}
	}
	records = append(records, record)

	path, err := historyPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Load reads all history records from disk. Returns an empty slice if the file
// does not yet exist.
func Load() ([]Record, error) {
	path, err := historyPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, err
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// PrintSummary prints a formatted table of all historical runs plus cumulative totals.
func PrintSummary(records []Record) {
	if len(records) == 0 {
		fmt.Println("No history found. Run nmod-cleaner to get started!")
		return
	}

	var totalDirs int
	var totalFailed int
	var totalBytes int64

	fmt.Println("\n--- Stats History ---")
	fmt.Printf("%-30s  %8s  %8s  %12s\n", "Date", "Cleaned", "Failed", "Freed")
	fmt.Println("─────────────────────────────────────────────────────")

	for _, r := range records {
		fmt.Printf("%-30s  %8d  %8d  %12s\n",
			r.Timestamp.Local().Format("2006-01-02 15:04:05"),
			r.DirsClean,
			r.DirsFailed,
			analyzer.FormatBytes(r.BytesFreed),
		)
		totalDirs += r.DirsClean
		totalFailed += r.DirsFailed
		totalBytes += r.BytesFreed
	}

	fmt.Println("─────────────────────────────────────────────────────")
	fmt.Printf("%-30s  %8d  %8d  %12s\n",
		fmt.Sprintf("TOTAL (%d runs)", len(records)),
		totalDirs,
		totalFailed,
		analyzer.FormatBytes(totalBytes),
	)
}
