// Package notify shows best-effort desktop notifications for background runs.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Failure shows a desktop notification. Errors are ignored: a notification
// must never fail a job run, and logs remain the source of truth.
func Failure(title string, message string) {
	if runtime.GOOS != "darwin" {
		return
	}
	script := fmt.Sprintf("display notification %q with title %q", message, title)
	_ = exec.Command("osascript", "-e", script).Run()
}
