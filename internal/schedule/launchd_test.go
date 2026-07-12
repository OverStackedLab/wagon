package schedule

import (
	"strings"
	"testing"
	"time"
)

func TestLabelSanitizesJobName(t *testing.T) {
	if got := Label("My Photos Backup!"); got != "dev.overstacked.wagon.job.my-photos-backup" {
		t.Fatalf("Label() = %q", got)
	}
}

func TestPlistPathIsUnderLaunchAgents(t *testing.T) {
	path, err := PlistPath("backup")
	if err != nil {
		t.Fatalf("PlistPath() error: %v", err)
	}
	if !strings.Contains(path, "Library/LaunchAgents/dev.overstacked.wagon.job.backup.plist") {
		t.Fatalf("PlistPath() = %q", path)
	}
}

func TestRenderPlist(t *testing.T) {
	plist := RenderPlist("photos & docs", "/usr/local/bin/wagon", "/tmp/wagon.log", 90*time.Minute)

	for _, want := range []string{
		"<string>dev.overstacked.wagon.job.photos---docs</string>",
		"<string>/usr/local/bin/wagon</string>",
		"<string>jobs</string>",
		"<string>run</string>",
		"<string>photos &amp; docs</string>",
		"<string>--scheduled</string>",
		"<integer>5400</integer>",
		"<key>RunAtLoad</key>",
		"<string>/tmp/wagon.log</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("RenderPlist() output is missing %q:\n%s", want, plist)
		}
	}
}
