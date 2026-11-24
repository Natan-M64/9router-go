package tokensaver

import (
	"strings"
	"testing"
)

// Ordinary text that reaches RTK with no resolved command (an MCP result, a
// fetched page, a file read through a wrapper) must not be claimed by a filter
// written for one CLI's output. Each of these used to be: the filters matched a
// single loose pattern ("failed", "INFO", any line shaped like a path, "Already
// on") and then kept only the lines they recognise, about a sixth of the text.
var ordinaryTexts = map[string]string{
	"markdown with failed and INFO": "# Notes\n\nThe first attempt failed, so we retried.\n" +
		"INFO is a log level, and ERROR is another.\n\n- item one\n- item two\n" +
		"See /usr/local/bin and ./scripts/run.sh for details.\n",
	"changelog mentioning a branch": "## 1.2.0\n\n* Already on the new branch model\n* Switched the default\n" +
		"* fixed the failed upload\n",
	"source file with paths": "package main\n\n// reads ./config/app.yaml and /etc/hosts\nfunc main() {\n\tprintln(\"failed\")\n}\n",
	"web page text": "Welcome. Deployment failed last night; INFO: services are back.\n" +
		"Docs live at /docs/index and ./guides/start.\n",
}

func TestContentSelectionIgnoresOrdinaryText(t *testing.T) {
	cfg := DefaultRTKConfig()
	for name, text := range ordinaryTexts {
		f := MatchFilter(text, "", cfg)
		if f != nil && f.ID != "generic-output" {
			t.Errorf("%s: content selected the %s filter", name, f.ID)
		}
	}
}

// Real output carries more than one sign of its command, and a filter with an
// unmistakable single signature opts out of the rule. Command selection is not
// affected at all.
func TestContentSelectionStillFindsRealCommandOutput(t *testing.T) {
	cfg := DefaultRTKConfig()
	cases := map[string]struct{ text, want string }{
		"git status": {"On branch main\nnothing to commit, working tree clean\n", "git-status"},
		"git diff":   {"diff --git a/f.go b/f.go\n--- a/f.go\n+++ b/f.go\n@@ -1,3 +1,3 @@\n-old\n+new\n", "git-diff"},
	}
	for name, c := range cases {
		if f := MatchFilter(c.text, "", cfg); f == nil || f.ID != c.want {
			t.Errorf("%s: filter = %v, want %s", name, f, c.want)
		}
	}
	// With the command known, the filter is selected by it, as before.
	if f := MatchFilter("X build  main  push  1m\n", "gh pr checks", cfg); f == nil || f.ID != "gh" {
		t.Errorf("gh not selected by its command: %v", f)
	}
	if f := MatchFilter(strings.Repeat("a.go\n", 5), "find . -name a.go", cfg); f == nil || f.ID != "shell-find" {
		t.Errorf("shell-find not selected by its command: %v", f)
	}
}

// The current-branch marker of `git branch` is "* name" alone on its line. A
// markdown bullet has more words after the star and is not that.
func TestGitBranchMarkerIsNotAMarkdownBullet(t *testing.T) {
	cfg := DefaultRTKConfig()
	bullets := "* Already on the new branch model\n* Switched the default\n"
	if f := MatchFilter(bullets, "", cfg); f != nil && f.ID == "git-branch" {
		t.Error("markdown bullets selected the git-branch filter")
	}
	real := "  dev\n* main\n  feature/x\nSwitched to branch 'main'\n"
	if f := MatchFilter(real, "", cfg); f == nil || f.ID != "git-branch" {
		t.Errorf("real branch output no longer selects git-branch: %v", f)
	}
	detached := "* (HEAD detached at 3f2a1bc)\n  main\nAlready on 'main'\n"
	if f := MatchFilter(detached, "", cfg); f == nil || f.ID != "git-branch" {
		t.Errorf("detached-HEAD output no longer selects git-branch: %v", f)
	}
}

// `go test` prints "ok" and "FAIL" in exactly that case. The filter's pattern
// was compiled case-insensitive, so prose that begins a line with "fail" counted
// as a second sign of Go test output.
func TestGoTestFilterNeedsTheExactStatusWord(t *testing.T) {
	cfg := DefaultRTKConfig()
	prose := "--- FAIL: this heading appears in a changelog entry\nfail the test rather than pass it.\n"
	if f := MatchFilter(prose, "", cfg); f != nil && f.ID == "test-go" {
		t.Error("prose selected the test-go filter")
	}
	real := "--- FAIL: TestX (0.00s)\nFAIL\tpkg/x\t0.01s\n"
	if f := MatchFilter(real, "", cfg); f == nil || f.ID != "test-go" {
		t.Errorf("real go test failure no longer selects test-go: %v", f)
	}
}

// Selection patterns are signatures of a command's output, and a CLI prints its
// status words in one exact case, so they are matched exactly. The shared
// compiler stays case-insensitive for the include and drop lists, and a
// selection pattern that wants case folding asks for it with (?i).
func TestSelectionPatternsAreCaseSensitive(t *testing.T) {
	exact := compileSelectionList([]string{`^FAIL\s+`, `(?i)^warning:`})
	if len(exact) != 2 {
		t.Fatalf("compiled %d patterns, want 2", len(exact))
	}
	if exact[0].MatchString("fail the test") {
		t.Error("selection pattern matched the wrong case")
	}
	if !exact[0].MatchString("FAIL\tpkg") {
		t.Error("selection pattern missed its own case")
	}
	if !exact[1].MatchString("WARNING: x") {
		t.Error("an explicit (?i) was not honoured")
	}
	if !compileRegexList([]string{`^FAIL\s+`})[0].MatchString("fail the test") {
		t.Error("the shared list compiler must stay case-insensitive")
	}
}
