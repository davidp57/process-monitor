package textdiff

import (
	"reflect"
	"strings"
	"testing"
)

func TestLines(t *testing.T) {
	a := "options = {\r\n\tgraphics = {\r\n\t\tvisibRange = \"Medium\",\r\n\t\tmsaa = 2,\r\n\t},\r\n}\r\n"
	b := "options = {\n\tgraphics = {\n\t\tvisibRange = \"High\",\n\t\tmsaa = 2,\n\t\tssaa = 1,\n\t},\n}\n"
	got := Lines(a, b)
	want := []Op{
		{'=', "options = {"}, {'=', "\tgraphics = {"},
		{'-', "\t\tvisibRange = \"Medium\","}, {'+', "\t\tvisibRange = \"High\","},
		{'=', "\t\tmsaa = 2,"}, {'+', "\t\tssaa = 1,"},
		{'=', "\t},"}, {'=', "}"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q", got)
	}
	if got := Lines("", "x\n"); !reflect.DeepEqual(got, []Op{{'+', "x"}}) {
		t.Errorf("création : %q", got)
	}
	if got := Lines("x\n", ""); !reflect.DeepEqual(got, []Op{{'-', "x"}}) {
		t.Errorf("suppression : %q", got)
	}
}

func TestHunksKeepContext(t *testing.T) {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line")
	}
	a := strings.Join(lines, "\n")
	lines[50] = "changed"
	b := strings.Join(lines, "\n")
	got := Hunks(Lines(a, b), 2)
	want := []Op{{'~', "48"}, {'=', "line"}, {'=', "line"}, {'-', "line"}, {'+', "changed"}, {'=', "line"}, {'=', "line"}, {'~', "47"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q", got)
	}
}

func TestBinary(t *testing.T) {
	if Binary([]byte("text"), 100) || !Binary([]byte("a\x00b"), 100) || !Binary(make([]byte, 101), 100) {
		t.Error("détection")
	}
}
