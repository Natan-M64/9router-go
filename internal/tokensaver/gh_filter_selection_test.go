package tokensaver

import "testing"

// Content matching must not hand ordinary text to the GitHub CLI filter. Its
// failure marker "X name" was compiled case-insensitive and unanchored, so it
// matched any word ending in x followed by a space ("textbox "Email"", "index
// 4", "linux amd64"), and the filter then kept only the lines it recognizes,
// leaving a line or two of the whole output.
func TestGitHubCLIFilterIsNotSelectedByWordsEndingInX(t *testing.T) {
	for _, text := range []string{
		"- textbox \"Email\" [ref=e11]\n- button \"Login\" [ref=e12]\n",
		"index 4 of the list\nsecond line of output\n",
		"linux amd64 build finished\nnothing else here\n",
	} {
		if f := MatchFilter(text, "", DefaultRTKConfig()); f != nil && f.ID == "gh" {
			t.Errorf("gh filter selected for %q", text)
		}
	}
	// The marker the pattern was written for still selects the filter.
	cli := "X build-and-test  main  push  1m\n✓ lint  main  push  30s\n"
	if f := MatchFilter(cli, "", DefaultRTKConfig()); f == nil || f.ID != "gh" {
		t.Fatalf("a real gh failure line no longer selects the gh filter: %v", f)
	}
}

// RTK selects a filter before it strips ANSI, and gh may indent or colour the
// marker, so the anchored pattern must still find it.
func TestGitHubCLIFilterSelectsIndentedAndColouredFailureMarker(t *testing.T) {
	for _, cli := range []string{
		"  X build-and-test  main  push  1m\n",
		"\x1b[31mX\x1b[0m build-and-test  main  push  1m\n",
		"\x1b[31mX build-and-test\x1b[0m  main  push  1m\n",
	} {
		if f := MatchFilter(cli, "", DefaultRTKConfig()); f == nil || f.ID != "gh" {
			t.Errorf("gh filter not selected for %q: %v", cli, f)
		}
	}
}
