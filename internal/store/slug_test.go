package store

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"My Cool Site":       "my-cool-site",
		"  Spaces  Around  ": "spaces-around",
		"Weird!!@#Chars":     "weird-chars",
		"already-slug":       "already-slug",
		"":                   "site",
		"---":                "site",
		"Café Déjà":          "caf-d-j",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
