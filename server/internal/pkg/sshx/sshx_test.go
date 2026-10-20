package sshx

import "testing"

func TestQuote(t *testing.T) {
	cases := map[string]string{
		"plain":          `'plain'`,
		"with space":     `'with space'`,
		"it's here":      `'it'\''s here'`,
		"$(rm -rf /)":    `'$(rm -rf /)'`,
		"a';rm -rf /;'b": `'a'\'';rm -rf /;'\''b'`,
		"back`tick`":     "'back`tick`'",
		`double"quote"`:  `'double"quote"'`,
		"中文/emoji🚀":      `'中文/emoji🚀'`,
	}
	for in, want := range cases {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %q, want %q", in, got, want)
		}
	}
}
