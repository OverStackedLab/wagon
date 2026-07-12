//go:build !unix

package jobs

// Without a portable liveness check, never treat a lock as stale.
func processAlive(pid int) bool {
	return true
}
