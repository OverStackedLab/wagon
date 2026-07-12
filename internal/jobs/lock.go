package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// acquireLock creates an exclusive per-job lock file and returns a release
// function. A lock held by a process that no longer exists is treated as
// stale and taken over, so a crashed run cannot block scheduled runs forever.
func acquireLock(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(file, "%d\n", os.Getpid())
			file.Close()
			return func() { os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("create lock file: %w", err)
		}

		pid, readErr := readLockPID(path)
		if readErr == nil && pid > 0 && processAlive(pid) {
			return nil, fmt.Errorf("job already running (pid %d holds %s)", pid, path)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove stale lock file: %w", err)
		}
	}
	return nil, fmt.Errorf("could not acquire lock %s", path)
}

func readLockPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
