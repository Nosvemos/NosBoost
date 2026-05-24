package cleaner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CleanupMetrics holds the results of the deep garbage collection sweep.
type CleanupMetrics struct {
	FilesDeleted int
	BytesFreed   int64
	FilesSkipped int
}

// safeDelete attempts to delete a file, catching locked errors gracefully.
// Returns the file size if deleted, 0 if skipped, and an optional error.
func safeDelete(path string) (int64, bool) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, false
	}

	// We only accumulate bytes for regular files (not symlinks or directories)
	var size int64
	if info.Mode().IsRegular() {
		size = info.Size()
	}

	err = os.Remove(path)
	if err != nil {
		// File is locked or access is denied
		return 0, false
	}

	return size, true
}

// wipeDirectory recursively traverses a folder and deletes all contents safely.
func wipeDirectory(dir string, metrics *CleanupMetrics) {
	// Guard against empty path or root partition deletion
	if dir == "" || len(dir) <= 3 {
		return
	}

	// We walk the directory
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			metrics.FilesSkipped++
			return nil // Skip and continue walk
		}

		if path == dir {
			return nil // Do not delete the target root directory itself
		}

		// Skip directories during the first file deletion pass
		if d.IsDir() {
			return nil
		}

		size, deleted := safeDelete(path)
		if deleted {
			metrics.FilesDeleted++
			metrics.BytesFreed += size
		} else {
			metrics.FilesSkipped++
		}

		return nil
	})

	// Perform a second pass to remove empty subdirectories
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == dir {
			return nil
		}
		if d.IsDir() {
			// Remove folder (will only succeed if folder is completely empty)
			_ = os.Remove(path)
		}
		return nil
	})
}

// scanAndCleanLogs traverses a folder recursively and safely deletes all '*.log' files.
func scanAndCleanLogs(dir string, metrics *CleanupMetrics) {
	if dir == "" || len(dir) <= 3 {
		return
	}

	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".log") {
			size, deleted := safeDelete(path)
			if deleted {
				metrics.FilesDeleted++
				metrics.BytesFreed += size
			} else {
				metrics.FilesSkipped++
			}
		}

		return nil
	})
}

// ExecuteDeepCleanup scans temporary directories, prefetch pools, error caches,
// and deletes junk logs safely, returning the detailed metrics of the sweep.
func ExecuteDeepCleanup() *CleanupMetrics {
	metrics := &CleanupMetrics{}

	// 1. WIPE USER TEMP (%TEMP%)
	userTemp := os.Getenv("TEMP")
	if userTemp != "" {
		wipeDirectory(userTemp, metrics)
	}

	// 2. WIPE SYSTEM TEMP (C:\Windows\Temp)
	systemTemp := os.Getenv("SystemRoot") + `\Temp`
	if _, err := os.Stat(systemTemp); err == nil {
		wipeDirectory(systemTemp, metrics)
	}

	// 3. WIPE WINDOWS PREFETCH (C:\Windows\Prefetch)
	prefetch := os.Getenv("SystemRoot") + `\Prefetch`
	if _, err := os.Stat(prefetch); err == nil {
		wipeDirectory(prefetch, metrics)
	}

	// 4. WIPE WINDOWS ERROR REPORTING DUMPS (%LOCALAPPDATA%\CrashDumps)
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		crashDumps := filepath.Join(localAppData, "CrashDumps")
		if _, err := os.Stat(crashDumps); err == nil {
			wipeDirectory(crashDumps, metrics)
		}
	}

	// 5. NUKES *.LOG FILES IN SCOPES
	if userTemp != "" {
		scanAndCleanLogs(userTemp, metrics)
	}
	if localAppData != "" {
		scanAndCleanLogs(localAppData, metrics)
	}

	return metrics
}
