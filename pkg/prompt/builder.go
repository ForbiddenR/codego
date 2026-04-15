package prompt

import (
	"fmt"
	"strings"
)

// Section is a named block of system prompt content.
type Section struct {
	Name    string
	Content string
	Weight  int // higher = later in prompt (controls ordering)
}

// Builder constructs a system prompt from ordered sections.
type Builder struct {
	sections []Section
}

// New creates a new prompt builder.
func New() *Builder {
	return &Builder{}
}

// Add appends a section to the prompt.
func (b *Builder) Add(name, content string) *Builder {
	if content == "" {
		return b
	}
	b.sections = append(b.sections, Section{Name: name, Content: content})
	return b
}

// AddWeighted adds a section with ordering weight.
func (b *Builder) AddWeighted(name, content string, weight int) *Builder {
	if content == "" {
		return b
	}
	b.sections = append(b.sections, Section{Name: name, Content: content, Weight: weight})
	return b
}

// Build assembles all sections into a single system prompt string.
func (b *Builder) Build() string {
	if len(b.sections) == 0 {
		return ""
	}

	// Sort by weight
	sorted := make([]Section, len(b.sections))
	copy(sorted, b.sections)
	sortByWeight(sorted)

	var sb strings.Builder
	for i, s := range sorted {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("<%s>\n%s\n</%s>", s.Name, s.Content, s.Name))
	}
	return sb.String()
}

// String is an alias for Build().
func (b *Builder) String() string {
	return b.Build()
}

// Sections returns all sections (unsorted).
func (b *Builder) Sections() []Section {
	return b.sections
}

// Count returns the number of sections.
func (b *Builder) Count() int {
	return len(b.sections)
}

// Clear removes all sections.
func (b *Builder) Clear() *Builder {
	b.sections = nil
	return b
}

func sortByWeight(sections []Section) {
	for i := 1; i < len(sections); i++ {
		for j := i; j > 0 && sections[j].Weight < sections[j-1].Weight; j-- {
			sections[j], sections[j-1] = sections[j-1], sections[j]
		}
	}
}
