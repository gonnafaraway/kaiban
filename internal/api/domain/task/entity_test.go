package task_test

import (
	"testing"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/domain/task"
)

func TestOverlayDoesNotDropBase(t *testing.T) {
	overlay := "be concise"
	col := &column.Column{
		SystemPromptTemplate: "base",
		UserCustomPrompt:     &overlay,
	}
	got := col.BuildSystemPrompt()
	if got != "base\n\n# User overlay\nbe concise" {
		t.Fatalf("unexpected overlay: %q", got)
	}
	col.ResetOverlay()
	if col.BuildSystemPrompt() != "base" {
		t.Fatal("reset overlay must keep template")
	}
}

func TestApproveRequiresSucceeded(t *testing.T) {
	tsk, err := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if err := tsk.Approve(false, uuid.New()); err != task.ErrNotSucceeded {
		t.Fatalf("got %v", err)
	}
}

func TestApproveMovesForward(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.MarkQueued()
	_ = tsk.MarkRunning()
	_ = tsk.MarkSucceeded("ok")
	next := uuid.New()
	if err := tsk.Approve(false, next); err != nil {
		t.Fatal(err)
	}
	if tsk.ColumnID != next || tsk.ExecutionStatus != task.StatusIdle {
		t.Fatalf("unexpected state %+v", tsk)
	}
}

func TestApproveLastColumnDone(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.MarkQueued()
	_ = tsk.MarkRunning()
	_ = tsk.MarkSucceeded("ok")
	if err := tsk.Approve(true, uuid.Nil); err != nil {
		t.Fatal(err)
	}
	if tsk.ExecutionStatus != task.StatusDone {
		t.Fatalf("got %s", tsk.ExecutionStatus)
	}
}

func TestReturnToPreviousOnly(t *testing.T) {
	curID, prevID := uuid.New(), uuid.New()
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, curID, uuid.New())
	current := &column.Column{ID: curID, OrderIndex: 2}
	later := &column.Column{ID: uuid.New(), OrderIndex: 3}
	if err := tsk.ReturnTo(current, later, "back"); err != task.ErrReturnTarget {
		t.Fatalf("forward return must fail, got %v", err)
	}
	prev := &column.Column{ID: prevID, OrderIndex: 1}
	if err := tsk.ReturnTo(current, prev, "fix"); err != nil {
		t.Fatal(err)
	}
	if tsk.ColumnID != prevID || tsk.ExecutionStatus != task.StatusIdle {
		t.Fatalf("unexpected %+v", tsk)
	}
}

func TestReturnRequiresComment(t *testing.T) {
	cur := &column.Column{ID: uuid.New(), OrderIndex: 1}
	prev := &column.Column{ID: uuid.New(), OrderIndex: 0}
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, cur.ID, uuid.New())
	if err := tsk.ReturnTo(cur, prev, "  "); err != task.ErrCommentRequired {
		t.Fatalf("got %v", err)
	}
}

func TestCannotRunWhileQueued(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.MarkQueued()
	if err := tsk.CanRun(); err != task.ErrAlreadyRunning {
		t.Fatalf("got %v", err)
	}
}

func TestFailStaysInColumn(t *testing.T) {
	col := uuid.New()
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, col, uuid.New())
	_ = tsk.MarkQueued()
	_ = tsk.MarkRunning()
	if err := tsk.MarkFailed("boom"); err != nil {
		t.Fatal(err)
	}
	if tsk.ColumnID != col || tsk.ExecutionStatus != task.StatusFailed {
		t.Fatalf("fail must stay in column: %+v", tsk)
	}
}

func TestArchiveSetsTimestamp(t *testing.T) {
	tsk, _ := task.New("t", "ctx", nil, task.Artifacts{JiraIssue: "P-1"}, uuid.New(), uuid.New())
	if tsk.IsArchived() {
		t.Fatal("new task must not be archived")
	}
	if err := tsk.Archive(); err != nil {
		t.Fatal(err)
	}
	if !tsk.IsArchived() || tsk.ArchivedAt == nil {
		t.Fatal("archive must set archived_at")
	}
}

func TestCannotArchiveTwice(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.Archive()
	if err := tsk.Archive(); err != task.ErrAlreadyArchived {
		t.Fatalf("got %v", err)
	}
}

func TestCannotArchiveWhileRunning(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.MarkQueued()
	if err := tsk.Archive(); err != task.ErrAlreadyRunning {
		t.Fatalf("got %v", err)
	}
}

func TestCannotRunArchived(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.Archive()
	if err := tsk.CanRun(); err != task.ErrArchived {
		t.Fatalf("got %v", err)
	}
	if err := tsk.Approve(false, uuid.New()); err != task.ErrArchived {
		t.Fatalf("got %v", err)
	}
}

func TestUnarchiveRestoresBoard(t *testing.T) {
	tsk, _ := task.New("t", "", nil, task.Artifacts{}, uuid.New(), uuid.New())
	_ = tsk.Archive()
	if err := tsk.Unarchive(); err != nil {
		t.Fatal(err)
	}
	if tsk.IsArchived() {
		t.Fatal("unarchive must clear archived_at")
	}
	if err := tsk.Unarchive(); err != task.ErrNotArchived {
		t.Fatalf("got %v", err)
	}
}
