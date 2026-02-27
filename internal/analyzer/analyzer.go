package analyzer

import (
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"sync"
)

// CalculateDirectorySize recursively computes total size of a path structure
func CalculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	err := filepath.WalkDir(dirPath, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable paths
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

// CalculateSizesConcurrently uses goroutines to process multiple paths
func CalculateSizesConcurrently(paths []string) map[string]int64 {
	resultsMap := make(map[string]int64)
	var mapMutex sync.Mutex
	var wg sync.WaitGroup

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			size, err := CalculateDirectorySize(p)
			if err == nil {
				mapMutex.Lock()
				resultsMap[p] = size
				mapMutex.Unlock()
			}
		}(path)
	}

	wg.Wait()
	return resultsMap
}

// FormatBytes returns a human readable representation of bytes
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 Bytes"
	}
	const k = 1024
	sizes := []string{"Bytes", "KB", "MB", "GB", "TB"}
	i := math.Floor(math.Log(float64(bytes)) / math.Log(k))
	val := float64(bytes) / math.Pow(k, i)
	return fmt.Sprintf("%.2f %s", val, sizes[int(i)])
}
