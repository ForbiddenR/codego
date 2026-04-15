package mcp

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m.Count() != 0 {
		t.Errorf("count = %d, want 0", m.Count())
	}
}

func TestManager_ListServers_Empty(t *testing.T) {
	m := NewManager()
	servers := m.ListServers()
	if len(servers) != 0 {
		t.Errorf("servers = %d, want 0", len(servers))
	}
}

func TestManager_ListAllTools_Empty(t *testing.T) {
	m := NewManager()
	tools := m.ListAllTools()
	if len(tools) != 0 {
		t.Errorf("tools = %d, want 0", len(tools))
	}
}

func TestManager_GetServer_Missing(t *testing.T) {
	m := NewManager()
	_, ok := m.GetServer("nonexistent")
	if ok {
		t.Error("should not find missing server")
	}
}

func TestManager_Disconnect_Missing(t *testing.T) {
	m := NewManager()
	err := m.Disconnect("nonexistent")
	if err == nil {
		t.Error("should error on missing server")
	}
}

func TestParseMCPToolName(t *testing.T) {
	tests := []struct {
		input        string
		wantServer   string
		wantTool     string
		wantOK       bool
	}{
		{"mcp__filesystem__read_file", "filesystem", "read_file", true},
		{"mcp__db__query", "db", "query", true},
		{"mcp__a__b__c", "a", "b__c", true},
		{"bash", "", "", false},
		{"mcp_incomplete", "", "", false},
		{"mcp__nosuffix", "", "", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		server, tool, ok := ParseMCPToolName(tt.input)
		if ok != tt.wantOK {
			t.Errorf("ParseMCPToolName(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			continue
		}
		if ok {
			if server != tt.wantServer {
				t.Errorf("server = %q, want %q", server, tt.wantServer)
			}
			if tool != tt.wantTool {
				t.Errorf("tool = %q, want %q", tool, tt.wantTool)
			}
		}
	}
}

func TestToolWrapper_Naming(t *testing.T) {
	m := NewManager()
	w := NewToolWrapper("fs", "read", "Read a file", nil, m)

	if w.Name() != "mcp__fs__read" {
		t.Errorf("name = %q, want %q", w.Name(), "mcp__fs__read")
	}
	if w.Description() != "[MCP:fs] Read a file" {
		t.Errorf("desc = %q", w.Description())
	}
}

func TestToolWrapper_Execute_NoServer(t *testing.T) {
	m := NewManager()
	w := NewToolWrapper("missing", "tool", "desc", nil, m)

	result, _ := w.Execute(nil, nil)
	if !result.IsError {
		t.Error("should be error when server not connected")
	}
}
