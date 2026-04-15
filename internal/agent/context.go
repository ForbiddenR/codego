package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectContext holds detected information about the working directory.
type ProjectContext struct {
	WorkingDir  string
	IsGitRepo   bool
	GitBranch   string
	GitRemote   string
	Language    string
	Framework   string
	ProjectName string
	HasClaudeMd bool
	ClaudeMd    string
	Files       []string // key config files found
}

// DetectProjectContext examines the working directory and returns context info.
func DetectProjectContext(workDir string) *ProjectContext {
	ctx := &ProjectContext{WorkingDir: workDir}

	detectGit(ctx, workDir)
	detectLanguage(ctx, workDir)
	detectClaudeMd(ctx, workDir)

	return ctx
}

// BuildSystemPrompt generates a system prompt from the project context.
func (ctx *ProjectContext) BuildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are CodeGo, an AI coding assistant. You help with software development tasks including writing code, debugging, testing, and explaining concepts.\n\n")

	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Be concise and direct\n")
	sb.WriteString("- Show code when relevant, explain when helpful\n")
	sb.WriteString("- Use the provided tools to read, write, and execute code\n")
	sb.WriteString("- Always verify your changes work\n")

	if ctx.Language != "" {
		sb.WriteString("\nProject context:\n")
		sb.WriteString("- Language: " + ctx.Language + "\n")
	}
	if ctx.Framework != "" {
		sb.WriteString("- Framework: " + ctx.Framework + "\n")
	}
	if ctx.IsGitRepo {
		sb.WriteString("- Git repo: yes")
		if ctx.GitBranch != "" {
			sb.WriteString(" (branch: " + ctx.GitBranch + ")")
		}
		sb.WriteString("\n")
	}
	if ctx.ProjectName != "" {
		sb.WriteString("- Project: " + ctx.ProjectName + "\n")
	}
	if len(ctx.Files) > 0 {
		sb.WriteString("- Key files: " + strings.Join(ctx.Files, ", ") + "\n")
	}

	if ctx.HasClaudeMd && ctx.ClaudeMd != "" {
		sb.WriteString("\n<project_instructions>\n")
		sb.WriteString(ctx.ClaudeMd)
		sb.WriteString("\n</project_instructions>\n")
	}

	return sb.String()
}

func detectGit(ctx *ProjectContext, dir string) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	if out, err := cmd.Output(); err == nil && strings.TrimSpace(string(out)) == "true" {
		ctx.IsGitRepo = true

		// Get branch
		branchCmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
		if out, err := branchCmd.Output(); err == nil {
			ctx.GitBranch = strings.TrimSpace(string(out))
		}

		// Get remote
		remoteCmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
		if out, err := remoteCmd.Output(); err == nil {
			ctx.GitRemote = strings.TrimSpace(string(out))
		}
	}
}

func detectLanguage(ctx *ProjectContext, dir string) {
	// Check for language indicators in order of specificity
	indicators := []struct {
		file      string
		language  string
		framework string
	}{
		{"go.mod", "Go", ""},
		{"Cargo.toml", "Rust", ""},
		{"package.json", "JavaScript/Node.js", detectJSFramework(dir)},
		{"pyproject.toml", "Python", detectPythonFramework(dir)},
		{"requirements.txt", "Python", ""},
		{"pom.xml", "Java", "Maven"},
		{"build.gradle", "Java", "Gradle"},
		{"Gemfile", "Ruby", ""},
		{"composer.json", "PHP", ""},
		{"CMakeLists.txt", "C/C++", "CMake"},
		{"Makefile", "C/C++", ""},
	}

	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(dir, ind.file)); err == nil {
			ctx.Language = ind.language
			ctx.Framework = ind.framework
			ctx.Files = append(ctx.Files, ind.file)
			break
		}
	}

	// Get project name from directory
	ctx.ProjectName = filepath.Base(dir)
}

func detectJSFramework(dir string) string {
	packageJSON, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	content := string(packageJSON)

	checks := []struct {
		pattern   string
		framework string
	}{
		{"next", "Next.js"},
		{"react", "React"},
		{"vue", "Vue.js"},
		{"express", "Express"},
		{"@nestjs/core", "NestJS"},
		{"astro", "Astro"},
		{"svelte", "Svelte"},
	}

	for _, c := range checks {
		if strings.Contains(content, c.pattern) {
			return c.framework
		}
	}
	return ""
}

func detectPythonFramework(dir string) string {
	// Check pyproject.toml
	if content, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
		s := string(content)
		if strings.Contains(s, "django") {
			return "Django"
		}
		if strings.Contains(s, "fastapi") {
			return "FastAPI"
		}
		if strings.Contains(s, "flask") {
			return "Flask"
		}
	}

	// Check requirements.txt
	if content, err := os.ReadFile(filepath.Join(dir, "requirements.txt")); err == nil {
		s := strings.ToLower(string(content))
		if strings.Contains(s, "django") {
			return "Django"
		}
		if strings.Contains(s, "fastapi") {
			return "FastAPI"
		}
		if strings.Contains(s, "flask") {
			return "Flask"
		}
	}
	return ""
}

func detectClaudeMd(ctx *ProjectContext, dir string) {
	// Check for CLAUDE.md or .claude/CLAUDE.md
	candidates := []string{
		filepath.Join(dir, "CLAUDE.md"),
		filepath.Join(dir, ".claude", "CLAUDE.md"),
		filepath.Join(dir, "claude.md"),
	}

	for _, path := range candidates {
		if content, err := os.ReadFile(path); err == nil {
			ctx.HasClaudeMd = true
			ctx.ClaudeMd = strings.TrimSpace(string(content))
			ctx.Files = append(ctx.Files, filepath.Base(path))
			return
		}
	}
}
