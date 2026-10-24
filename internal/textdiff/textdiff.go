// Package textdiff compares two versions of a text file line by line, for
// the wizard's report: what a profile changed in options.lua.
package textdiff

import (
	"bytes"
	"strings"
)

// Op is one line of a diff.
type Op struct {
	Kind byte   `json:"kind"` // '=' kept, '-' removed, '+' added
	Text string `json:"text"`
}

// Binary reports whether data is better shown by its size than by its
// lines: a NUL byte in its start, or more than limit bytes.
func Binary(data []byte, limit int) bool {
	head := data
	if len(head) > 8192 {
		head = head[:8192]
	}
	return len(data) > limit || bytes.IndexByte(head, 0) >= 0
}

// maxCells bounds the comparison table of the part that differs: past it,
// that part is shown as all removed then all added.
const maxCells = 4_000_000

// Lines compares a and b. The lines both share at their start and end are
// kept as they are, and only the part between is compared, so a few
// changed lines in a long file cost little.
func Lines(a, b string) []Op {
	x, y := split(a), split(b)
	pre := 0
	for pre < len(x) && pre < len(y) && x[pre] == y[pre] {
		pre++
	}
	suf := 0
	for suf < len(x)-pre && suf < len(y)-pre && x[len(x)-1-suf] == y[len(y)-1-suf] {
		suf++
	}
	var out []Op
	for _, l := range x[:pre] {
		out = append(out, Op{'=', l})
	}
	out = append(out, middle(x[pre:len(x)-suf], y[pre:len(y)-suf])...)
	for _, l := range x[len(x)-suf:] {
		out = append(out, Op{'=', l})
	}
	return out
}

func split(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// middle is the longest common subsequence of x and y, as operations.
func middle(x, y []string) []Op {
	var out []Op
	if len(x)*len(y) > maxCells {
		for _, l := range x {
			out = append(out, Op{'-', l})
		}
		for _, l := range y {
			out = append(out, Op{'+', l})
		}
		return out
	}
	// lcs[i][j]: length of the common subsequence of x[i:] and y[j:].
	lcs := make([][]int32, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int32, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		switch {
		case x[i] == y[j]:
			out = append(out, Op{'=', x[i]})
			i, j = i+1, j+1
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, Op{'-', x[i]})
			i++
		default:
			out = append(out, Op{'+', y[j]})
			j++
		}
	}
	for ; i < len(x); i++ {
		out = append(out, Op{'-', x[i]})
	}
	for ; j < len(y); j++ {
		out = append(out, Op{'+', y[j]})
	}
	return out
}

// Hunks keeps the changed lines with context lines around them; the kept
// lines further away are dropped, a run of them replaced by one Op with
// Kind '~' whose Text is how many were skipped.
func Hunks(ops []Op, context int) []Op {
	keep := make([]bool, len(ops))
	for i, op := range ops {
		if op.Kind != '=' {
			for k := max(0, i-context); k <= min(len(ops)-1, i+context); k++ {
				keep[k] = true
			}
		}
	}
	var out []Op
	skipped := 0
	for i, op := range ops {
		if keep[i] {
			if skipped > 0 {
				out = append(out, Op{'~', itoa(skipped)})
				skipped = 0
			}
			out = append(out, op)
		} else {
			skipped++
		}
	}
	if skipped > 0 {
		out = append(out, Op{'~', itoa(skipped)})
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
