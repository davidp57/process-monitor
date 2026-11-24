package affinity

import (
	"math/bits"
	"testing"

	"github.com/davidp57/process-monitor/internal/i18n"
)

func TestParse(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Mask
		text string
	}{
		{"0-11", 0x0FFF, "0-11"},
		{"12-15", 0xF000, "12-15"},
		{"5", 1 << 5, "5"},
		{" 0-5, 8 ,10-11 ", 0b1101_0011_1111, "0-5,8,10-11"},
		{"3,1,2", 0b1110, "1-3"},
		{"0-63", ^Mask(0), "0-63"},
	} {
		got, err := Parse(c.in)
		if err != nil || got != c.want {
			t.Errorf("Parse(%q) = %#x, %v ; attendu %#x", c.in, got, err, c.want)
			continue
		}
		if s := got.String(); s != c.text {
			t.Errorf("%#x.String() = %q ; attendu %q", got, s, c.text)
		}
	}
}

func TestParseRejects(t *testing.T) {
	i18n.Set("fr")
	for _, in := range []string{"", " ", "a", "12-11", "0-", "-3", "1,,2", "64", "0-64", "0x0FFF", "1.5"} {
		if m, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %#x, attendu une erreur", in, m)
		}
	}
}

func TestSystemListsThisMachinesCores(t *testing.T) {
	m, err := System()
	if err != nil {
		t.Fatal(err)
	}
	if m == 0 || m&1 == 0 {
		t.Fatalf("System() = %#x : au moins le cœur 0 attendu", m)
	}
	if n := bits.OnesCount64(uint64(m)); n < 1 {
		t.Fatalf("System() = %#x", m)
	}
}
