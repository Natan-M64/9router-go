package tokensaver

import (
	"regexp"
	"strings"
)

// ariaNodeRe matches one node line of an accessibility-tree snapshot, the YAML
// -like text Playwright and the browser MCP servers return for a page:
//
//   - button "Login" [ref=e5]
//   - img [ref=e6]
var ariaNodeRe = regexp.MustCompile(`^( *)- ([A-Za-z][\w-]*)(.*)$`)

// ariaRefRe is the element reference an agent passes back to click or type. A
// snapshot without references is not one an agent acts on, so it is left alone.
var ariaRefRe = regexp.MustCompile(`\[ref=e?\d+\]`)

// minARIARefs is how many referenced nodes a text needs before it is treated as
// a snapshot. A markdown list that happens to contain "- item" lines has none.
const minARIARefs = 5

// ariaDecorativeRoles are the roles that carry no meaning when they have no
// name, text or state of their own.
var ariaDecorativeRoles = map[string]bool{
	"img":          true,
	"generic":      true,
	"presentation": true,
	"none":         true,
}

type ariaLine struct {
	node   bool
	indent int
	role   string
	rest   string
}

// compactARIASnapshot shrinks an accessibility-tree snapshot before the generic
// head/tail truncation sees it: it drops leaf nodes that say nothing, repeats
// that until no wrapper is left empty, and halves a two-space indentation.
//
// It never drops a node an agent could act on or read: a role outside the
// decorative set, a quoted name, text after the colon, or any attribute other
// than the element reference (cursor=pointer, checked, disabled, ...) keeps the
// node. Lines that are not nodes (page URL, console messages, code fences) pass
// through unchanged.
//
// The result is stable under repetition. The indentation is halved only while
// the smallest step is exactly two, so a second pass finds a step of one and
// leaves it. That matters because the RTK stage and the built-in Headroom
// engine run the same fold.
func compactARIASnapshot(text string) (string, bool) {
	if strings.Count(text, "[ref=") < minARIARefs {
		return text, false
	}
	raw := strings.Split(text, "\n")
	lines := make([]ariaLine, len(raw))
	refs := 0
	for i, l := range raw {
		m := ariaNodeRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		lines[i] = ariaLine{node: true, indent: len(m[1]), role: m[2], rest: m[3]}
		if ariaRefRe.MatchString(m[3]) {
			refs++
		}
	}
	if refs < minARIARefs {
		return text, false
	}

	removed := make([]bool, len(raw))
	for pass := 0; pass < 8; pass++ {
		changed := false
		for i := range lines {
			if removed[i] || !lines[i].node || !ariaDecorativeRoles[lines[i].role] {
				continue
			}
			if !ariaEmptyRest(lines[i].rest) || ariaHasLiveChild(lines, removed, i) {
				continue
			}
			removed[i] = true
			changed = true
		}
		if !changed {
			break
		}
	}

	halve := ariaSmallestStep(lines, removed) == 2
	out := make([]string, 0, len(raw))
	for i, l := range raw {
		if removed[i] {
			continue
		}
		if halve && lines[i].node {
			l = l[lines[i].indent/2:]
		}
		out = append(out, l)
	}
	next := strings.Join(out, "\n")
	if len(next) >= len(text) {
		return text, false
	}
	return next, true
}

// ariaEmptyRest reports whether what follows the role is nothing but the element
// reference and an optional trailing colon.
func ariaEmptyRest(rest string) bool {
	stripped := strings.TrimSpace(ariaRefRe.ReplaceAllString(rest, ""))
	return stripped == "" || stripped == ":"
}

// ariaHasLiveChild reports whether the node at i still has a kept child: the
// next kept line is a node indented deeper than it.
func ariaHasLiveChild(lines []ariaLine, removed []bool, i int) bool {
	for j := i + 1; j < len(lines); j++ {
		if removed[j] {
			continue
		}
		return lines[j].node && lines[j].indent > lines[i].indent
	}
	return false
}

// ariaSmallestStep is the smallest positive indentation among the kept nodes,
// or 0 when there is none.
func ariaSmallestStep(lines []ariaLine, removed []bool) int {
	step := 0
	for i, l := range lines {
		if removed[i] || !l.node || l.indent <= 0 {
			continue
		}
		if step == 0 || l.indent < step {
			step = l.indent
		}
	}
	return step
}
