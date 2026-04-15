package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nice-code/codego/internal/types"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpen(t *testing.T) {
	s := testStore(t)
	if s.db == nil {
		t.Error("db should not be nil")
	}
}

func TestOpen_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "sessions.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("db file should have been created")
	}
}

func TestCreateSession(t *testing.T) {
	s := testStore(t)

	err := s.CreateSession("test-1", "My Session")
	if err != nil {
		t.Fatalf("CreateSession() error: %v", err)
	}

	info, err := s.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() error: %v", err)
	}
	if info.ID != "test-1" {
		t.Errorf("id = %q, want %q", info.ID, "test-1")
	}
	if info.Title != "My Session" {
		t.Errorf("title = %q, want %q", info.Title, "My Session")
	}
	if info.MsgCount != 0 {
		t.Errorf("msg count = %d, want 0", info.MsgCount)
	}
}

func TestCreateSession_Duplicate(t *testing.T) {
	s := testStore(t)
	s.CreateSession("dup", "First")
	err := s.CreateSession("dup", "Second")
	if err == nil {
		t.Error("duplicate id should fail")
	}
}

func TestUpdateTitle(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Old Title")

	s.UpdateTitle("s1", "New Title")

	info, _ := s.GetSession("s1")
	if info.Title != "New Title" {
		t.Errorf("title = %q, want %q", info.Title, "New Title")
	}
}

func TestDeleteSession(t *testing.T) {
	s := testStore(t)
	s.CreateSession("del", "Delete Me")
	s.AppendMessage("del", types.NewUserMessage("hello"))

	s.DeleteSession("del")

	_, err := s.GetSession("del")
	if err == nil {
		t.Error("deleted session should not be found")
	}
}

func TestListSessions(t *testing.T) {
	s := testStore(t)

	time.Sleep(10 * time.Millisecond)
	s.CreateSession("s1", "First")
	time.Sleep(10 * time.Millisecond)
	s.CreateSession("s2", "Second")
	time.Sleep(10 * time.Millisecond)
	s.CreateSession("s3", "Third")

	sessions, err := s.ListSessions(0)
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("count = %d, want 3", len(sessions))
	}

	// Should be ordered by most recent
	if sessions[0].ID != "s3" {
		t.Errorf("first = %q, want s3", sessions[0].ID)
	}
	if sessions[2].ID != "s1" {
		t.Errorf("last = %q, want s1", sessions[2].ID)
	}
}

func TestListSessions_Limit(t *testing.T) {
	s := testStore(t)
	s.CreateSession("a", "A")
	s.CreateSession("b", "B")
	s.CreateSession("c", "C")

	sessions, _ := s.ListSessions(2)
	if len(sessions) != 2 {
		t.Errorf("count = %d, want 2", len(sessions))
	}
}

func TestAppendMessage(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Test")

	msg := types.NewUserMessage("hello world")
	err := s.AppendMessage("s1", msg)
	if err != nil {
		t.Fatalf("AppendMessage() error: %v", err)
	}

	count, _ := s.MessageCount("s1")
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestAppendMessage_Multiple(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Test")

	s.AppendMessage("s1", types.NewUserMessage("user msg"))
	s.AppendMessage("s1", types.NewAssistantText("assistant msg"))
	s.AppendMessage("s1", types.NewToolResultMessage("t1", "output", false))

	count, _ := s.MessageCount("s1")
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestGetMessages(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Test")

	s.AppendMessage("s1", types.NewUserMessage("first"))
	s.AppendMessage("s1", types.NewAssistantText("second"))
	s.AppendMessage("s1", types.NewUserMessage("third"))

	msgs, err := s.GetMessages("s1")
	if err != nil {
		t.Fatalf("GetMessages() error: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("count = %d, want 3", len(msgs))
	}
	if msgs[0].Role != types.RoleUser {
		t.Errorf("msg[0] role = %q", msgs[0].Role)
	}
	if msgs[0].TextContent() != "first" {
		t.Errorf("msg[0] text = %q", msgs[0].TextContent())
	}
	if msgs[1].Role != types.RoleAssistant {
		t.Errorf("msg[1] role = %q", msgs[1].Role)
	}
	if msgs[2].TextContent() != "third" {
		t.Errorf("msg[2] text = %q", msgs[2].TextContent())
	}
}

func TestGetMessages_Empty(t *testing.T) {
	s := testStore(t)
	s.CreateSession("empty", "Empty")

	msgs, _ := s.GetMessages("empty")
	if msgs != nil {
		t.Errorf("expected nil for empty session, got %d messages", len(msgs))
	}
}

func TestGetMessages_PreservesToolCalls(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Tool Test")

	toolMsg := types.NewAssistantMessage(
		types.NewTextBlock("let me run a command"),
		types.NewToolUseBlock("t1", "bash", map[string]interface{}{"command": "ls"}),
	)
	s.AppendMessage("s1", toolMsg)

	msgs, _ := s.GetMessages("s1")
	if len(msgs) != 1 {
		t.Fatalf("count = %d, want 1", len(msgs))
	}
	if !msgs[0].HasToolCalls() {
		t.Error("should have tool calls")
	}
	calls := msgs[0].ToolCalls()
	if calls[0].Name != "bash" {
		t.Errorf("tool name = %q", calls[0].Name)
	}
}

func TestExportJSONL(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Export")

	s.AppendMessage("s1", types.NewUserMessage("hi"))
	s.AppendMessage("s1", types.NewAssistantText("hello"))

	jsonl, err := s.ExportJSONL("s1")
	if err != nil {
		t.Fatalf("ExportJSONL() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(jsonl), "\n")
	if len(lines) != 2 {
		t.Errorf("lines = %d, want 2", len(lines))
	}

	// Each line should be valid JSON
	for _, line := range lines {
		if !strings.HasPrefix(line, "{") {
			t.Errorf("line should be JSON: %s", line)
		}
	}
}

func TestSearchSessions(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Go programming")
	s.CreateSession("s2", "Python scripts")
	s.CreateSession("s3", "Go testing")

	s.AppendMessage("s1", types.NewUserMessage("how to write Go tests"))
	s.AppendMessage("s2", types.NewUserMessage("python hello world"))

	results, err := s.SearchSessions("Go", 0)
	if err != nil {
		t.Fatalf("SearchSessions() error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results = %d, want 2", len(results))
	}
}

func TestSessionInfo_TouchesUpdated(t *testing.T) {
	s := testStore(t)
	s.CreateSession("s1", "Test")

	before, _ := s.GetSession("s1")
	time.Sleep(10 * time.Millisecond)
	s.AppendMessage("s1", types.NewUserMessage("msg"))
	after, _ := s.GetSession("s1")

	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Error("updated_at should be newer after append")
	}
}
