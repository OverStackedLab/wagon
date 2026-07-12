package browser

import (
	"context"
	"strings"
	"testing"

	"github.com/OverStackedLab/wagon/internal/filelist"
)

func TestToggleTransferPauseRequestsPause(t *testing.T) {
	model := Model{
		copying: true,
		transfer: transferState{
			items: []filelist.Item{{Name: "one.txt"}},
		},
	}

	cmd := model.toggleTransferPause()
	if cmd != nil {
		t.Fatal("toggleTransferPause() returned a command while requesting pause")
	}
	if !model.transfer.pauseRequested {
		t.Fatal("toggleTransferPause() did not request pause")
	}
	if model.transfer.paused {
		t.Fatal("toggleTransferPause() paused before the current item finished")
	}
}

func TestCopyStepFinishedPausesBeforeNextItem(t *testing.T) {
	model := Model{
		copying: true,
		transfer: transferState{
			items: []filelist.Item{
				{Name: "one.txt"},
				{Name: "two.txt"},
			},
			pauseRequested: true,
		},
	}

	updated, cmd := model.Update(copyStepFinishedMsg{itemIndex: 0})
	if cmd != nil {
		t.Fatal("Update() returned a command while pausing before next item")
	}

	got := updated.(Model)
	if !got.copying {
		t.Fatal("copying should remain true while the queue is paused")
	}
	if !got.transfer.paused {
		t.Fatal("queue was not paused before the next item")
	}
	if got.transfer.pauseRequested {
		t.Fatal("pause request should be cleared after queue pauses")
	}
	if got.transfer.current != 1 {
		t.Fatalf("current = %d, want 1", got.transfer.current)
	}
}

func TestToggleTransferPauseResumesPausedQueue(t *testing.T) {
	model := Model{
		copying: true,
		transfer: transferState{
			items:  []filelist.Item{{Name: "one.txt"}},
			paused: true,
		},
	}

	cmd := model.toggleTransferPause()
	if cmd == nil {
		t.Fatal("toggleTransferPause() did not return a resume command")
	}
	if model.transfer.paused {
		t.Fatal("toggleTransferPause() did not clear paused state")
	}
	if model.transfer.pauseRequested {
		t.Fatal("toggleTransferPause() should clear pause request when resuming")
	}
}

func TestVisibleSizeItemsSkipsKnownAndParentItems(t *testing.T) {
	p := pane{
		items: []filelist.Item{
			{Name: "..", Path: "/tmp", IsParent: true, IsDir: true},
			{Name: "Documents", Path: "/tmp/Documents", IsDir: true},
			{Name: "Photos", Path: "/tmp/Photos", IsDir: true, SizeKnown: true, Size: 1024},
			{Name: "Archive", Path: "/tmp/Archive", IsDir: true},
		},
	}

	items := visibleSizeItems(p)
	if len(items) != 2 {
		t.Fatalf("len(visibleSizeItems()) = %d, want 2", len(items))
	}
	if items[0].Name != "Documents" || items[1].Name != "Archive" {
		t.Fatalf("visibleSizeItems() = %v, want Documents and Archive", itemNames(items))
	}
}

func TestVisibleSizeItemsHonorsSearch(t *testing.T) {
	p := pane{
		search: "doc",
		items: []filelist.Item{
			{Name: "Documents", Path: "/tmp/Documents", IsDir: true},
			{Name: "Photos", Path: "/tmp/Photos", IsDir: true},
		},
	}

	items := visibleSizeItems(p)
	if len(items) != 1 {
		t.Fatalf("len(visibleSizeItems()) = %d, want 1", len(items))
	}
	if items[0].Name != "Documents" {
		t.Fatalf("visibleSizeItems()[0].Name = %q, want Documents", items[0].Name)
	}
}

func TestSizeCurrentSelectionRestartsRunningSizeJob(t *testing.T) {
	canceled := false
	model := Model{
		ctx:           context.Background(),
		sizing:        true,
		nextSizeJobID: 1,
		sizeJob: sizeState{
			id: 1,
			cancel: func() {
				canceled = true
			},
		},
		panes: [2]pane{
			{
				path: "/tmp",
				items: []filelist.Item{
					{Name: "Documents", Path: "/tmp/Documents", IsDir: true},
				},
				selected: map[string]bool{},
			},
		},
	}

	cmd := model.sizeCurrentSelection()
	if cmd == nil {
		t.Fatal("sizeCurrentSelection() did not start a replacement job")
	}
	if !canceled {
		t.Fatal("sizeCurrentSelection() did not cancel the existing size job")
	}
	if !model.sizing {
		t.Fatal("replacement size job is not marked as running")
	}
	if model.sizeJob.id != 2 {
		t.Fatalf("new size job id = %d, want 2", model.sizeJob.id)
	}
}

func TestStaleSizeResultIsIgnored(t *testing.T) {
	model := Model{
		sizing: true,
		sizeJob: sizeState{
			id:        2,
			paneIndex: 0,
			panePath:  "/tmp",
			paneKind:  filelist.Local,
		},
		panes: [2]pane{
			{
				kind: filelist.Local,
				path: "/tmp",
				items: []filelist.Item{
					{Name: "Documents", Path: "/tmp/Documents", IsDir: true},
				},
			},
		},
	}

	updated, cmd := model.Update(sizeStepFinishedMsg{
		jobID:     1,
		paneIndex: 0,
		paneKind:  filelist.Local,
		panePath:  "/tmp",
		itemPath:  "/tmp/Documents",
		size:      1024,
	})
	if cmd != nil {
		t.Fatal("Update() returned a command for a stale size result")
	}

	got := updated.(Model)
	if !got.sizing {
		t.Fatal("stale size result stopped the current size job")
	}
	if got.panes[0].items[0].SizeKnown {
		t.Fatal("stale size result updated the pane")
	}
}

func TestSizeTickAdvancesSpinner(t *testing.T) {
	model := Model{
		sizing: true,
		sizeJob: sizeState{
			items: []filelist.Item{{Name: "Documents"}},
		},
	}

	updated, cmd := model.Update(sizeTickMsg{})
	if cmd == nil {
		t.Fatal("Update() did not return another size tick command")
	}

	got := updated.(Model)
	if got.sizeJob.spinner != 1 {
		t.Fatalf("spinner = %d, want 1", got.sizeJob.spinner)
	}
}

func TestRenderSizeAnalysisShowsLoader(t *testing.T) {
	model := Model{
		sizing: true,
		sizeJob: sizeState{
			items: []filelist.Item{{Name: "Documents"}},
		},
	}

	got := model.renderSizeAnalysis(100)
	for _, want := range []string{"Analyzing sizes 1/1", "Documents", "Esc cancels"} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderSizeAnalysis() = %q, missing %q", got, want)
		}
	}
}

func itemNames(items []filelist.Item) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}
