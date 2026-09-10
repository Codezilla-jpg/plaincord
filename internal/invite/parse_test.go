package invite

import "testing"

func TestParse(t *testing.T) {
	cases := map[string]string{
		"abc":                                  "abc",
		"https://discord.gg/abc":               "abc",
		"https://discord.gg/abc/":              "abc",
		"discord.gg/abc":                       "abc",
		"https://discord.com/invite/xyz":       "xyz",
		"https://discord.com/invite/xyz?foo=1": "xyz",
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
	if _, err := Parse(""); err != ErrInvalid {
		t.Fatalf("empty: %v", err)
	}
	if _, err := Parse("https://example.com/abc"); err != ErrInvalid {
		t.Fatalf("foreign host: %v", err)
	}
}
