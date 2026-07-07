package browser

import (
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
