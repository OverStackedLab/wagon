// Package schedule manages recurring background job runs. On macOS it
// delegates to launchd via per-job LaunchAgents instead of running a daemon.
package schedule

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/OverStackedLab/wagon/internal/jobs"
)

const labelPrefix = "dev.overstacked.wagon.job."

// Label returns the launchd label for a job.
func Label(jobName string) string {
	return labelPrefix + jobs.SanitizeName(jobName)
}

// PlistPath returns the LaunchAgent plist path for a job.
func PlistPath(jobName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", Label(jobName)+".plist"), nil
}

// IsScheduled reports whether a LaunchAgent plist exists for the job.
func IsScheduled(jobName string) bool {
	path, err := PlistPath(jobName)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// RenderPlist builds the LaunchAgent XML that runs the job headlessly on an
// interval. launchd-level output (for example panics) goes to the same log
// the runner appends to.
func RenderPlist(jobName string, wagonPath string, logPath string, every time.Duration) string {
	programArgs := []string{wagonPath, "jobs", "run", jobName, "--scheduled"}

	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	fmt.Fprintf(&b, "\t<key>Label</key>\n\t<string>%s</string>\n", xmlEscape(Label(jobName)))
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range programArgs {
		fmt.Fprintf(&b, "\t\t<string>%s</string>\n", xmlEscape(arg))
	}
	b.WriteString("\t</array>\n")
	fmt.Fprintf(&b, "\t<key>StartInterval</key>\n\t<integer>%d</integer>\n", int(every.Seconds()))
	b.WriteString("\t<key>RunAtLoad</key>\n\t<false/>\n")
	fmt.Fprintf(&b, "\t<key>StandardOutPath</key>\n\t<string>%s</string>\n", xmlEscape(logPath))
	fmt.Fprintf(&b, "\t<key>StandardErrorPath</key>\n\t<string>%s</string>\n", xmlEscape(logPath))
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

// Install writes the LaunchAgent plist and loads it into the user's launchd
// domain, replacing any previous schedule for the job.
func Install(ctx context.Context, jobName string, wagonPath string, logPath string, every time.Duration) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("scheduling uses launchd and is only supported on macOS for now")
	}

	path, err := PlistPath(jobName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(RenderPlist(jobName, wagonPath, logPath, every)), 0o644); err != nil {
		return "", fmt.Errorf("write LaunchAgent plist: %w", err)
	}

	domain := fmt.Sprintf("gui/%d", os.Getuid())
	// Unload any previous version first so re-scheduling updates the interval.
	_ = exec.CommandContext(ctx, "launchctl", "bootout", domain+"/"+Label(jobName)).Run()

	output, err := exec.CommandContext(ctx, "launchctl", "bootstrap", domain, path).CombinedOutput()
	if err != nil {
		// Older macOS releases predate bootstrap; fall back to legacy load.
		legacyOutput, legacyErr := exec.CommandContext(ctx, "launchctl", "load", "-w", path).CombinedOutput()
		if legacyErr != nil {
			return "", fmt.Errorf("launchctl bootstrap failed: %v: %s (legacy load also failed: %v: %s)",
				err, bytes.TrimSpace(output), legacyErr, bytes.TrimSpace(legacyOutput))
		}
	}
	return path, nil
}

// Uninstall unloads the job's LaunchAgent and removes its plist.
func Uninstall(ctx context.Context, jobName string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("scheduling uses launchd and is only supported on macOS for now")
	}

	path, err := PlistPath(jobName)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("job %q is not scheduled", jobName)
	}

	domain := fmt.Sprintf("gui/%d", os.Getuid())
	_ = exec.CommandContext(ctx, "launchctl", "bootout", domain+"/"+Label(jobName)).Run()
	_ = exec.CommandContext(ctx, "launchctl", "unload", path).Run()

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("remove LaunchAgent plist: %w", err)
	}
	return path, nil
}

func xmlEscape(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}
