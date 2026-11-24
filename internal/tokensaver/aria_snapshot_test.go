package tokensaver

import (
	"strings"
	"testing"
)

const ariaSample = "### Page\n" +
	"- Page URL: https://example.test/\n" +
	"### Snapshot\n" +
	"```yaml\n" +
	"- generic [ref=e1]:\n" +
	"  - generic [ref=e2]:\n" +
	"    - img [ref=e3]\n" +
	"    - presentation [ref=e4]\n" +
	"    - generic [ref=e5]\n" +
	"  - navigation [ref=e6]:\n" +
	"    - link \"Home\" [ref=e7] [cursor=pointer]:\n" +
	"      - img [ref=e8]\n" +
	"    - generic [cursor=pointer] [ref=e9]\n" +
	"    - generic [ref=e10]: Welcome back\n" +
	"  - textbox \"Email\" [ref=e11]\n" +
	"  - button \"Login\" [ref=e12]\n" +
	"  - img \"logo\" [ref=e13]\n" +
	"```\n"

func TestCompactARIASnapshotDropsOnlyDecorativeLeaves(t *testing.T) {
	out, did := compactARIASnapshot(ariaSample)
	if !did {
		t.Fatal("snapshot was not compacted")
	}
	for _, gone := range []string{"[ref=e3]", "[ref=e4]", "[ref=e5]", "[ref=e8]"} {
		if strings.Contains(out, gone) {
			t.Errorf("decorative node %s survived:\n%s", gone, out)
		}
	}
	// Everything an agent can act on or read is still there with its reference.
	for _, kept := range []string{
		`link "Home" [ref=e7] [cursor=pointer]`,
		"generic [cursor=pointer] [ref=e9]",
		"generic [ref=e10]: Welcome back",
		`textbox "Email" [ref=e11]`,
		`button "Login" [ref=e12]`,
		`img "logo" [ref=e13]`,
		"- Page URL: https://example.test/",
		"```yaml",
	} {
		if !strings.Contains(out, kept) {
			t.Errorf("lost %q:\n%s", kept, out)
		}
	}
	if len(out) >= len(ariaSample) {
		t.Fatalf("no saving: %d -> %d", len(ariaSample), len(out))
	}
}

// A wrapper whose children were all decorative is itself empty and goes too.
func TestCompactARIASnapshotRemovesWrapperLeftEmpty(t *testing.T) {
	in := "- generic [ref=e1]:\n  - generic [ref=e2]:\n    - img [ref=e3]\n" +
		"- button \"a\" [ref=e4]\n- button \"b\" [ref=e5]\n- button \"c\" [ref=e6]\n"
	out, _ := compactARIASnapshot(in)
	if strings.Contains(out, "e1]") || strings.Contains(out, "e2]") || strings.Contains(out, "e3]") {
		t.Fatalf("empty wrapper chain survived:\n%s", out)
	}
}

func TestCompactARIASnapshotHalvesTwoSpaceIndent(t *testing.T) {
	out, _ := compactARIASnapshot(ariaSample)
	if !strings.Contains(out, "\n - navigation [ref=e6]:\n") {
		t.Fatalf("indent was not halved:\n%s", out)
	}
	if !strings.Contains(out, "\n  - link \"Home\"") {
		t.Fatalf("nested indent was not halved:\n%s", out)
	}
}

// The RTK stage and the built-in Headroom engine run the same fold, so a second
// pass must be a no-op rather than halving the indentation again.
func TestCompactARIASnapshotIsIdempotent(t *testing.T) {
	once, _ := compactARIASnapshot(ariaSample)
	twice, did := compactARIASnapshot(once)
	if did || twice != once {
		t.Fatalf("second pass changed the text (did=%v):\n%s", did, twice)
	}
}

// Text that is not a snapshot is returned untouched, even when it is full of
// "- item" lines.
func TestCompactARIASnapshotIgnoresOrdinaryLists(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("  - generic item\n")
	}
	if out, did := compactARIASnapshot(sb.String()); did || out != sb.String() {
		t.Fatalf("a plain list was rewritten (did=%v)", did)
	}
}

// Through the public entry point the technique is reported and the references
// an agent needs are intact.
func TestCompressTextAppliesARIASnapshot(t *testing.T) {
	res := CompressTextDetailedWithCommand(ariaSample, "", DefaultRTKConfig())
	if !strings.Contains(strings.Join(res.TechniquesUsed, ","), "aria-snapshot") {
		t.Fatalf("aria-snapshot technique not applied: %v", res.TechniquesUsed)
	}
	if !strings.Contains(res.Text, `button "Login" [ref=e12]`) {
		t.Fatalf("actionable node lost:\n%s", res.Text)
	}
}
