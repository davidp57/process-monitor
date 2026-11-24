// Package affinity reads core lists ("0-11", "0-5,8") and applies them as
// process affinity masks: the cores a process may run on.
//
// A mask covers the first 64 logical cores, the processor group Windows
// gives pm and the programs it starts.
package affinity

import (
	"math/bits"
	"strconv"
	"strings"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Mask has bit n set when logical core n is allowed.
type Mask uint64

// maxCore is the highest core a mask can name.
const maxCore = 63

// Parse reads a core list: cores and ranges separated by commas, "0-11",
// "0-5,8,10-11". The result is never empty.
func Parse(s string) (Mask, error) {
	if strings.TrimSpace(s) == "" {
		return 0, i18n.Errorf("affinity.empty")
	}
	var m Mask
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		lo, hi, isRange := strings.Cut(part, "-")
		first, err1 := core(lo)
		last, err2 := first, error(nil)
		if isRange {
			last, err2 = core(hi)
		}
		if err1 != nil || err2 != nil || last < first {
			return 0, i18n.Errorf("affinity.bad_part", part, s)
		}
		for n := first; n <= last; n++ {
			m |= 1 << n
		}
	}
	return m, nil
}

func core(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 || n > maxCore {
		return 0, strconv.ErrRange
	}
	return n, nil
}

// String writes the mask back as a core list, ranges merged: "0-11".
func (m Mask) String() string {
	var parts []string
	for n := 0; n <= maxCore; {
		if m&(1<<n) == 0 {
			n++
			continue
		}
		end := n
		for end < maxCore && m&(1<<(end+1)) != 0 {
			end++
		}
		if end == n {
			parts = append(parts, strconv.Itoa(n))
		} else {
			parts = append(parts, strconv.Itoa(n)+"-"+strconv.Itoa(end))
		}
		n = end + 1
	}
	return strings.Join(parts, ",")
}

// Count is the number of cores in the mask.
func (m Mask) Count() int { return bits.OnesCount64(uint64(m)) }
