package commands

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r.Count() != 0 {
		t.Errorf("count = %d, want 0", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	r.Register(&Command{Name: "test", Description: "A test command"})
	if r.Count() != 1 {
		t.Errorf("count = %d, want 1", r.Count())
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&Command{Name: "test", Description: "Test"})

	cmd, ok := r.Get("test")
	if !ok {
		t.Fatal("should find command")
	}
	if cmd.Name != "test" {
		t.Errorf("name = %q", cmd.Name)
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("should not find missing command")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&Command{Name: "zbra", Description: "last"})
	r.Register(&Command{Name: "alpha", Description: "first"})

	cmds := r.List()
	if len(cmds) != 2 {
		t.Fatalf("count = %d, want 2", len(cmds))
	}
	if cmds[0].Name != "alpha" {
		t.Errorf("first = %q, want alpha", cmds[0].Name)
	}
	if cmds[1].Name != "zbra" {
		t.Errorf("second = %q, want zbra", cmds[1].Name)
	}
}

func TestRegistry_FindMatches(t *testing.T) {
	r := NewRegistry()
	r.Register(&Command{Name: "help"})
	r.Register(&Command{Name: "hello"})
	r.Register(&Command{Name: "model"})

	matches := r.FindMatches("hel")
	if len(matches) != 2 {
		t.Errorf("matches = %d, want 2", len(matches))
	}

	single := r.FindMatches("mod")
	if len(single) != 1 || single[0] != "model" {
		t.Errorf("single match = %v", single)
	}

	none := r.FindMatches("xyz")
	if len(none) != 0 {
		t.Errorf("no match = %v", none)
	}
}

func TestRegistry_Execute(t *testing.T) {
	r := NewRegistry()
	executed := false
	r.Register(&Command{
		Name: "test",
		Handler: func(args []string, ctx *Context) error {
			executed = true
			return nil
		},
	})

	err := r.Execute("/test", &Context{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !executed {
		t.Error("handler should have been called")
	}
}

func TestRegistry_Execute_WithArgs(t *testing.T) {
	r := NewRegistry()
	var gotArgs []string
	r.Register(&Command{
		Name: "echo",
		Handler: func(args []string, ctx *Context) error {
			gotArgs = args
			return nil
		},
	})

	r.Execute("/echo hello world", &Context{})
	if len(gotArgs) != 2 || gotArgs[0] != "hello" || gotArgs[1] != "world" {
		t.Errorf("args = %v, want [hello world]", gotArgs)
	}
}

func TestRegistry_Execute_Unknown(t *testing.T) {
	r := NewRegistry()
	err := r.Execute("/nonexistent", &Context{})
	if err == nil {
		t.Error("should error on unknown command")
	}
}

func TestRegistry_Execute_Ambiguous(t *testing.T) {
	r := NewRegistry()
	r.Register(&Command{Name: "help", Handler: func(_ []string, _ *Context) error { return nil }})
	r.Register(&Command{Name: "hello", Handler: func(_ []string, _ *Context) error { return nil }})

	err := r.Execute("/hel", &Context{})
	if err == nil {
		t.Error("should error on ambiguous prefix")
	}
}

func TestRegistry_Execute_Prefix(t *testing.T) {
	r := NewRegistry()
	called := false
	r.Register(&Command{Name: "model", Handler: func(_ []string, _ *Context) error {
		called = true
		return nil
	}})

	r.Execute("/mod", &Context{})
	if !called {
		t.Error("unambiguous prefix should work")
	}
}

func TestRegistry_Execute_NotACommand(t *testing.T) {
	r := NewRegistry()
	err := r.Execute("not a command", &Context{})
	if err == nil {
		t.Error("should error when input doesn't start with /")
	}
}

// ─── Default Commands Tests ───

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()
	if r.Count() == 0 {
		t.Error("default registry should have commands")
	}

	// Check key commands exist
	required := []string{"help", "model", "compact", "cost", "status", "reset", "save", "quit", "reasoning"}
	for _, name := range required {
		if _, ok := r.Get(name); !ok {
			t.Errorf("missing command: /%s", name)
		}
	}
}

func TestCommand_Model_Show(t *testing.T) {
	r := DefaultRegistry()
	ctx := &Context{Model: "claude-opus"}
	err := r.Execute("/model", ctx)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestCommand_Model_Change(t *testing.T) {
	r := DefaultRegistry()
	ctx := &Context{Model: "old-model"}
	r.Execute("/model new-model", ctx)
	if ctx.Model != "new-model" {
		t.Errorf("model = %q, want new-model", ctx.Model)
	}
}

func TestCommand_Reasoning(t *testing.T) {
	r := DefaultRegistry()
	err := r.Execute("/reasoning high", &Context{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestCommand_Reasoning_Invalid(t *testing.T) {
	r := DefaultRegistry()
	err := r.Execute("/reasoning turbo", &Context{})
	if err == nil {
		t.Error("should error on invalid level")
	}
}

func TestCommand_Reset(t *testing.T) {
	r := DefaultRegistry()
	called := false
	ctx := &Context{OnReset: func() { called = true }}
	r.Execute("/reset", ctx)
	if !called {
		t.Error("OnReset should have been called")
	}
}

func TestCommand_Cost(t *testing.T) {
	r := DefaultRegistry()
	ctx := &Context{
		SessionID: "test-123",
		MsgCount:  5,
		InputTok:  1000,
		OutputTok: 500,
	}
	err := r.Execute("/cost", ctx)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestCommand_Quit(t *testing.T) {
	r := DefaultRegistry()
	err := r.Execute("/quit", &Context{})
	if err != ErrQuit {
		t.Errorf("should return ErrQuit, got: %v", err)
	}
}

func TestCommand_Quit_Alias(t *testing.T) {
	r := DefaultRegistry()

	err := r.Execute("/exit", &Context{})
	if err != ErrQuit {
		t.Errorf("/exit should return ErrQuit")
	}

	err = r.Execute("/q", &Context{})
	if err != ErrQuit {
		t.Errorf("/q should return ErrQuit")
	}
}

func TestCommand_Compact(t *testing.T) {
	r := DefaultRegistry()
	called := false
	ctx := &Context{OnCompact: func() error { called = true; return nil }}
	r.Execute("/compact", ctx)
	if !called {
		t.Error("OnCompact should have been called")
	}
}

func TestCommand_Save(t *testing.T) {
	r := DefaultRegistry()
	var savedPath string
	ctx := &Context{OnSave: func(p string) error { savedPath = p; return nil }}
	r.Execute("/save test.jsonl", ctx)
	if savedPath != "test.jsonl" {
		t.Errorf("path = %q, want test.jsonl", savedPath)
	}
}
