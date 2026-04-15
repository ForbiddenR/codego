package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nice-code/codego/internal/types"
)

// ─── Registry Tests ───

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(NewFileReadTool())

	tool, ok := r.Get("read")
	if !ok {
		t.Fatal("expected to find 'read' tool")
	}
	if tool.Name() != "read" {
		t.Errorf("name = %q, want %q", tool.Name(), "read")
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent tool")
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()
	r.Register(NewFileReadTool())
	r.Register(NewFileWriteTool())
	r.Register(NewFileEditTool())

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() count = %d, want 3", len(all))
	}
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry()
	if r.Count() != 0 {
		t.Errorf("empty registry count = %d, want 0", r.Count())
	}
	r.Register(NewFileReadTool())
	if r.Count() != 1 {
		t.Errorf("count = %d, want 1", r.Count())
	}
}

func TestRegistry_ToolDefs(t *testing.T) {
	r := NewRegistry()
	r.Register(NewFileReadTool())
	r.Register(NewBashTool("/tmp"))

	defs := r.ToolDefs()
	if len(defs) != 2 {
		t.Fatalf("ToolDefs count = %d, want 2", len(defs))
	}

	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	if !names["read"] || !names["bash"] {
		t.Errorf("missing expected tool names: %v", names)
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()
	r.Register(NewFileReadTool())
	r.Register(NewFileWriteTool())

	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("Names() count = %d, want 2", len(names))
	}
}

func TestValidate(t *testing.T) {
	schema := types.NewObjectSchema(
		"test",
		map[string]*types.JSONSchema{
			"name": types.NewStringSchema("name"),
			"age":  types.NewNumberSchema("age"),
		},
		"name",
	)

	// Missing required field
	err := Validate(types.ToolInput{"age": 30}, schema)
	if err == nil {
		t.Error("should fail with missing required field")
	}

	// All required fields present
	err = Validate(types.ToolInput{"name": "test", "age": 30}, schema)
	if err != nil {
		t.Errorf("should pass: %v", err)
	}

	// Nil schema
	err = Validate(types.ToolInput{}, nil)
	if err != nil {
		t.Errorf("nil schema should pass: %v", err)
	}
}

// ─── BashTool Tests ───

func TestBashTool_ExecEcho(t *testing.T) {
	tool := NewBashTool("/tmp")
	result, err := tool.Execute(context.Background(), types.ToolInput{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("should not be error: %s", result.Output)
	}
	if result.Output != "hello" {
		t.Errorf("output = %q, want %q", result.Output, "hello")
	}
}

func TestBashTool_EmptyCommand(t *testing.T) {
	tool := NewBashTool("/tmp")
	result, _ := tool.Execute(context.Background(), types.ToolInput{})
	if !result.IsError {
		t.Error("empty command should be error")
	}
}

func TestBashTool_FailingCommand(t *testing.T) {
	tool := NewBashTool("/tmp")
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"command": "exit 1",
	})
	if !result.IsError {
		t.Error("failing command should be error")
	}
}

func TestBashTool_NoOutput(t *testing.T) {
	tool := NewBashTool("/tmp")
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"command": "true",
	})
	if result.Output != "(no output)" {
		t.Errorf("output = %q, want %q", result.Output, "(no output)")
	}
}

func TestBashTool_Timeout(t *testing.T) {
	tool := NewBashTool("/tmp")
	tool.Timeout = 100 * time.Millisecond
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"command": "sleep 10",
	})
	if !result.IsError {
		t.Error("timed out command should be error")
	}
	if !strings.Contains(result.Output, "timed out") {
		t.Errorf("should mention timeout: %s", result.Output)
	}
}

func TestBashTool_Cwd(t *testing.T) {
	tool := NewBashTool("/tmp")
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"command": "pwd",
	})
	if !strings.Contains(result.Output, "/tmp") {
		t.Errorf("should run in /tmp: %s", result.Output)
	}
}

func TestBashTool_Schema(t *testing.T) {
	tool := NewBashTool("/tmp")
	schema := tool.InputSchema()
	if schema.Type != "object" {
		t.Errorf("schema type = %q, want %q", schema.Type, "object")
	}
	if len(schema.Required) != 1 || schema.Required[0] != "command" {
		t.Errorf("required = %v, want [command]", schema.Required)
	}
}

// ─── FileReadTool Tests ───

func TestFileReadTool_ReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644)

	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"path": path})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}
	if !strings.Contains(result.Output, "line1") {
		t.Errorf("should contain line1: %s", result.Output)
	}
	if !strings.Contains(result.Output, "1\t") {
		t.Errorf("should have line numbers: %s", result.Output)
	}
}

func TestFileReadTool_WithOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0o644)

	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":   path,
		"offset": float64(3),
	})

	if !strings.Contains(result.Output, "c") {
		t.Errorf("should start at line 3: %s", result.Output)
	}
	if strings.Contains(result.Output, "a") {
		t.Errorf("should not contain line 1: %s", result.Output)
	}
}

func TestFileReadTool_WithLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("1\n2\n3\n4\n5\n"), 0o644)

	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":  path,
		"limit": float64(2),
	})

	if !strings.Contains(result.Output, "2\t") {
		t.Errorf("should contain line 2: %s", result.Output)
	}
	if strings.Contains(result.Output, "4\t") {
		t.Errorf("should not contain line 4: %s", result.Output)
	}
}

func TestFileReadTool_MissingFile(t *testing.T) {
	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"path": "/nonexistent/file"})
	if !result.IsError {
		t.Error("missing file should be error")
	}
}

func TestFileReadTool_Directory(t *testing.T) {
	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"path": "/tmp"})
	if !result.IsError {
		t.Error("directory should be error")
	}
}

func TestFileReadTool_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin.dat")
	os.WriteFile(path, []byte{0x00, 0x01, 0x02, 0xff}, 0o644)

	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"path": path})
	if !result.IsError {
		t.Error("binary file should be error")
	}
}

func TestFileReadTool_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte{}, 0o644)

	tool := NewFileReadTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"path": path})
	if result.Output != "(empty file)" {
		t.Errorf("output = %q, want %q", result.Output, "(empty file)")
	}
}

// ─── FileWriteTool Tests ───

func TestFileWriteTool_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	tool := NewFileWriteTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":    path,
		"content": "hello world",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Created") {
		t.Errorf("should say Created: %s", result.Output)
	}

	// Verify file content
	content, _ := os.ReadFile(path)
	if string(content) != "hello world" {
		t.Errorf("file content = %q, want %q", string(content), "hello world")
	}
}

func TestFileWriteTool_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	os.WriteFile(path, []byte("old"), 0o644)

	tool := NewFileWriteTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":    path,
		"content": "new",
	})

	if !strings.Contains(result.Output, "Overwrote") {
		t.Errorf("should say Overwrote: %s", result.Output)
	}
}

func TestFileWriteTool_CreateDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "file.txt")

	tool := NewFileWriteTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":    path,
		"content": "deep",
	})

	if result.IsError {
		t.Fatalf("should create dirs: %s", result.Output)
	}

	content, _ := os.ReadFile(path)
	if string(content) != "deep" {
		t.Errorf("content = %q, want %q", string(content), "deep")
	}
}

// ─── FileEditTool Tests ───

func TestFileEditTool_Replace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	tool := NewFileEditTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":       path,
		"old_string": "world",
		"new_string": "golang",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Replaced 1") {
		t.Errorf("should say replaced 1: %s", result.Output)
	}

	content, _ := os.ReadFile(path)
	if string(content) != "hello golang" {
		t.Errorf("content = %q, want %q", string(content), "hello golang")
	}
}

func TestFileEditTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	tool := NewFileEditTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":       path,
		"old_string": "xyz",
		"new_string": "abc",
	})

	if !result.IsError {
		t.Error("should be error when old_string not found")
	}
}

func TestFileEditTool_MultipleOccurrences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	os.WriteFile(path, []byte("aaa bbb aaa"), 0o644)

	tool := NewFileEditTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":       path,
		"old_string": "aaa",
		"new_string": "ccc",
	})

	if !result.IsError {
		t.Error("should fail with multiple occurrences without replace_all")
	}
}

func TestFileEditTool_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	os.WriteFile(path, []byte("aaa bbb aaa"), 0o644)

	tool := NewFileEditTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"path":        path,
		"old_string":  "aaa",
		"new_string":  "ccc",
		"replace_all": true,
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}

	content, _ := os.ReadFile(path)
	if string(content) != "ccc bbb ccc" {
		t.Errorf("content = %q, want %q", string(content), "ccc bbb ccc")
	}
}

// ─── GlobTool Tests ───

func TestGlobTool_FindFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte{}, 0o644)

	tool := NewGlobTool(dir)
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "*.go",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}
	if !strings.Contains(result.Output, "a.go") {
		t.Errorf("should find a.go: %s", result.Output)
	}
	if !strings.Contains(result.Output, "b.go") {
		t.Errorf("should find b.go: %s", result.Output)
	}
	if strings.Contains(result.Output, "c.txt") {
		t.Errorf("should not find c.txt: %s", result.Output)
	}
}

func TestGlobTool_NoMatches(t *testing.T) {
	tool := NewGlobTool("/tmp")
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "nonexistent_*.xyz",
	})
	if !strings.Contains(result.Output, "No files") {
		t.Errorf("should report no files: %s", result.Output)
	}
}

func TestGlobTool_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "deep.go"), []byte{}, 0o644)

	tool := NewGlobTool(dir)
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "**/*.go",
	})

	if !strings.Contains(result.Output, "deep.go") {
		t.Errorf("should find deep.go recursively: %s", result.Output)
	}
}

// ─── GrepTool Tests ───

func TestGrepTool_FindMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n"), 0o644)

	tool := NewGrepTool(dir)
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "func main",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}
	if !strings.Contains(result.Output, "func main()") {
		t.Errorf("should find func main: %s", result.Output)
	}
	if !strings.Contains(result.Output, "code.go:3") {
		t.Errorf("should show file and line: %s", result.Output)
	}
}

func TestGrepTool_WithContext(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("line1\nline2\ntarget\nline4\nline5\n"), 0o644)

	tool := NewGrepTool(dir)
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "target",
		"context": float64(1),
	})

	if !strings.Contains(result.Output, "line2") {
		t.Errorf("should include context before: %s", result.Output)
	}
	if !strings.Contains(result.Output, "line4") {
		t.Errorf("should include context after: %s", result.Output)
	}
}

func TestGrepTool_IncludeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("match here"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("match here too"), 0o644)

	tool := NewGrepTool(dir)
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "match",
		"include": "*.go",
	})

	if !strings.Contains(result.Output, "a.go") {
		t.Errorf("should find in .go: %s", result.Output)
	}
	if strings.Contains(result.Output, "b.txt") {
		t.Errorf("should not find in .txt: %s", result.Output)
	}
}

func TestGrepTool_NoMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("nothing here"), 0o644)

	tool := NewGrepTool(dir)
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "nonexistent_xyz",
	})

	if !strings.Contains(result.Output, "No matches") {
		t.Errorf("should report no matches: %s", result.Output)
	}
}

func TestGrepTool_InvalidRegex(t *testing.T) {
	tool := NewGrepTool("/tmp")
	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"pattern": "[invalid",
	})
	if !result.IsError {
		t.Error("invalid regex should be error")
	}
}

// ─── TodoWriteTool Tests ───

func TestTodoWriteTool_AddAndList(t *testing.T) {
	tool := NewTodoWriteTool()
	tool.Execute(context.Background(), types.ToolInput{
		"action": "add",
		"text":   "Build the thing",
	})

	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "list"})
	if !strings.Contains(result.Output, "Build the thing") {
		t.Errorf("should list task: %s", result.Output)
	}
	if !strings.Contains(result.Output, "[ ]") {
		t.Errorf("should show pending icon: %s", result.Output)
	}
}

func TestTodoWriteTool_UpdateStatus(t *testing.T) {
	tool := NewTodoWriteTool()
	tool.Execute(context.Background(), types.ToolInput{
		"action": "add",
		"text":   "Task 1",
	})

	result, _ := tool.Execute(context.Background(), types.ToolInput{
		"action": "update",
		"id":     float64(1),
		"status": "completed",
	})

	if !strings.Contains(result.Output, "completed") {
		t.Errorf("should show completed: %s", result.Output)
	}
}

func TestTodoWriteTool_Clear(t *testing.T) {
	tool := NewTodoWriteTool()
	tool.Execute(context.Background(), types.ToolInput{"action": "add", "text": "A"})
	tool.Execute(context.Background(), types.ToolInput{"action": "add", "text": "B"})

	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "clear"})
	if !strings.Contains(result.Output, "Cleared 2") {
		t.Errorf("should clear 2: %s", result.Output)
	}
}

func TestTodoWriteTool_UnknownAction(t *testing.T) {
	tool := NewTodoWriteTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "bad"})
	if !result.IsError {
		t.Error("unknown action should be error")
	}
}

func TestTodoWriteTool_EmptyList(t *testing.T) {
	tool := NewTodoWriteTool()
	result, _ := tool.Execute(context.Background(), types.ToolInput{"action": "list"})
	if !strings.Contains(result.Output, "No tasks") {
		t.Errorf("empty list: %s", result.Output)
	}
}
