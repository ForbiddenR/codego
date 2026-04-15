package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nice-code/codego/internal/tools"
	"github.com/nice-code/codego/internal/types"
)

func TestREPL_Run_Quit(t *testing.T) {
	mock := &mockAPI{}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	err := repl.Run(context.Background())
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	out := output.String()
	if !strings.Contains(out, "CodeGo") {
		t.Errorf("should show banner: %s", out)
	}
	if !strings.Contains(out, "Bye!") {
		t.Errorf("should show bye: %s", out)
	}
}

func TestREPL_Run_SingleQuery(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: textEvents("Hello there!")},
		},
	}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("hi\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	out := output.String()
	if !strings.Contains(out, "Hello there!") {
		t.Errorf("should contain response: %s", out)
	}
}

func TestREPL_Run_MultipleMessages(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: textEvents("Response 1")},
			{events: textEvents("Response 2")},
		},
	}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("msg1\nmsg2\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	out := output.String()
	if !strings.Contains(out, "Response 1") {
		t.Errorf("missing response 1: %s", out)
	}
	if !strings.Contains(out, "Response 2") {
		t.Errorf("missing response 2: %s", out)
	}
}

func TestREPL_Run_Help(t *testing.T) {
	mock := &mockAPI{}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("/help\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	out := output.String()
	if !strings.Contains(out, "/quit") {
		t.Errorf("help should list /quit: %s", out)
	}
	if !strings.Contains(out, "/model") {
		t.Errorf("help should list /model: %s", out)
	}
}

func TestREPL_Run_Reset(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: textEvents("response")},
		},
	}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("hello\n/reset\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	// After reset, messages should be cleared
	if len(a.Messages()) != 0 {
		t.Errorf("messages after reset = %d, want 0", len(a.Messages()))
	}
}

func TestREPL_Run_EmptyLines(t *testing.T) {
	mock := &mockAPI{}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("\n\n\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	err := repl.Run(context.Background())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestREPL_Run_UnknownCommand(t *testing.T) {
	mock := &mockAPI{}
	a := New(mock, tools.NewRegistry())

	input := strings.NewReader("/nonexistent\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	out := output.String()
	if !strings.Contains(out, "unknown command") {
		t.Errorf("should show error: %s", out)
	}
}

func TestREPL_Run_WithToolCalls(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: toolUseEvents("t1", "echo", `{"text": "hi"}`)},
			{events: textEvents("Tool result processed")},
		},
	}
	reg := tools.NewRegistry()
	reg.Register(&stubTool{name: "echo", response: types.NewToolResult("ok")})

	a := New(mock, reg, WithMaxIterations(5))

	input := strings.NewReader("run tool\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	out := output.String()
	if !strings.Contains(out, "Running echo") {
		t.Errorf("should show tool start: %s", out)
	}
	if !strings.Contains(out, "echo done") {
		t.Errorf("should show tool done: %s", out)
	}
	if !strings.Contains(out, "Tool result processed") {
		t.Errorf("should show response: %s", out)
	}
}

func TestREPL_RunSingle(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: textEvents("Single response")},
		},
	}
	a := New(mock, tools.NewRegistry())

	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Output: output,
	})

	err := repl.RunSingle(context.Background(), "explain main.go")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	out := output.String()
	if !strings.Contains(out, "Single response") {
		t.Errorf("should contain response: %s", out)
	}
}

func TestREPL_Compact(t *testing.T) {
	mock := &mockAPI{
		responses: []mockResponse{
			{events: textEvents("r1")},
			{events: textEvents("r2")},
			{events: textEvents("r3")},
			{events: textEvents("r4")},
			{events: textEvents("r5")},
		},
	}
	a := New(mock, tools.NewRegistry(), WithMaxIterations(5))

	input := strings.NewReader("a\nb\nc\nd\ne\n/compact\n/quit\n")
	output := &bytes.Buffer{}

	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Input:  input,
		Output: output,
	})

	repl.Run(context.Background())

	out := output.String()
	if !strings.Contains(out, "compressed") {
		t.Errorf("should show compression: %s", out)
	}
}

func TestREPL_Banner(t *testing.T) {
	mock := &mockAPI{}
	a := New(mock, tools.NewRegistry())

	output := &bytes.Buffer{}
	repl, _ := NewREPL(REPLConfig{
		Agent:  a,
		Output: output,
	})

	repl.printBanner()

	out := output.String()
	if !strings.Contains(out, "CodeGo") {
		t.Errorf("banner should contain CodeGo: %s", out)
	}
}

func TestREPL_ExitAliases(t *testing.T) {
	mock := &mockAPI{}
	a := New(mock, tools.NewRegistry())

	tests := []string{"/quit", "/exit", "/q"}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			input := strings.NewReader(cmd + "\n")
			output := &bytes.Buffer{}

			repl, _ := NewREPL(REPLConfig{
				Agent:  a,
				Input:  input,
				Output: output,
			})

			err := repl.Run(context.Background())
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !strings.Contains(output.String(), "Bye!") {
				t.Errorf("%s should show bye", cmd)
			}
		})
	}
}
