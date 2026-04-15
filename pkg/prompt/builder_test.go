package prompt

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	b := New()
	if b.Count() != 0 {
		t.Errorf("count = %d, want 0", b.Count())
	}
}

func TestBuilder_Add(t *testing.T) {
	b := New()
	b.Add("identity", "You are CodeGo")
	b.Add("tools", "You have access to tools")

	if b.Count() != 2 {
		t.Errorf("count = %d, want 2", b.Count())
	}
}

func TestBuilder_Add_Empty(t *testing.T) {
	b := New()
	b.Add("empty", "")
	if b.Count() != 0 {
		t.Errorf("empty content should not be added, count = %d", b.Count())
	}
}

func TestBuilder_Build(t *testing.T) {
	b := New()
	b.Add("identity", "You are CodeGo")
	b.Add("rules", "Be helpful")

	result := b.Build()

	if !strings.Contains(result, "<identity>") {
		t.Errorf("should contain <identity>: %s", result)
	}
	if !strings.Contains(result, "You are CodeGo") {
		t.Errorf("should contain content: %s", result)
	}
	if !strings.Contains(result, "</identity>") {
		t.Errorf("should contain closing tag: %s", result)
	}
	if !strings.Contains(result, "<rules>") {
		t.Errorf("should contain <rules>: %s", result)
	}
}

func TestBuilder_Build_Empty(t *testing.T) {
	b := New()
	if b.Build() != "" {
		t.Error("empty builder should return empty string")
	}
}

func TestBuilder_Build_WeightOrder(t *testing.T) {
	b := New()
	b.AddWeighted("high", "last", 10)
	b.AddWeighted("low", "first", 1)
	b.AddWeighted("mid", "middle", 5)

	result := b.Build()
	lowIdx := strings.Index(result, "first")
	midIdx := strings.Index(result, "middle")
	highIdx := strings.Index(result, "last")

	if lowIdx > midIdx || midIdx > highIdx {
		t.Errorf("weight ordering wrong:\n%s", result)
	}
}

func TestBuilder_String(t *testing.T) {
	b := New()
	b.Add("test", "hello")
	if b.String() != b.Build() {
		t.Error("String() should match Build()")
	}
}

func TestBuilder_Clear(t *testing.T) {
	b := New()
	b.Add("a", "content")
	b.Clear()
	if b.Count() != 0 {
		t.Errorf("count = %d after clear", b.Count())
	}
	if b.Build() != "" {
		t.Error("build should be empty after clear")
	}
}

func TestBuilder_Sections(t *testing.T) {
	b := New()
	b.Add("a", "1")
	b.Add("b", "2")
	sections := b.Sections()
	if len(sections) != 2 {
		t.Errorf("sections = %d, want 2", len(sections))
	}
}
