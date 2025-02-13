package internal

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed test/Changelog.md
var cl []byte

func TestTrimChangelog(t *testing.T) {
	cases := []struct {
		input    []byte
		trim     int
		expected int
	}{
		{cl, 50, 44},                    // default trim
		{cl, 0, 195},                    // do not trim if nothing is found
		{cl, 200, 195},                  // do not trim if less than expected
		{nil, 200, 1},                   // edge case
		{[]byte(``), 100, 1},            // edge case
		{[]byte(`# Changelog`), 100, 1}, // edge case
	}

	for _, c := range cases {
		cl := TrimChangelog(c.input, c.trim)
		t.Log(string(cl))
		got := len(strings.Split(string(cl), "\n"))
		if got != c.expected {
			t.Logf("got %d, expected %d", got, c.expected)
			t.Fail()
		}
	}
}
