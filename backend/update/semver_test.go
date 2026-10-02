package update

import "testing"

func TestParseVersion(t *testing.T) {
	validas := map[string]string{
		"1.2.3":         "1.2.3",
		"v1.2.3":        "1.2.3",
		" V2.0.0 ":      "2.0.0",
		"1.2":           "1.2.0",
		"0.0.0-dev":     "0.0.0-dev",
		"1.0.0-rc.1":    "1.0.0-rc.1",
		"1.0.0+build.7": "1.0.0",
	}
	for in, want := range validas {
		v, err := ParseVersion(in)
		if err != nil {
			t.Errorf("ParseVersion(%q) erro: %v", in, err)
			continue
		}
		if v.String() != want {
			t.Errorf("ParseVersion(%q) = %s, esperava %s", in, v, want)
		}
	}

	for _, in := range []string{"", "1", "a.b.c", "1.2.3.4", "1.-2.3", "1.2.3-", "1..3", "1.2.3-rc..1", "latest"} {
		if _, err := ParseVersion(in); err == nil {
			t.Errorf("ParseVersion(%q) deveria falhar", in)
		}
	}
}

func TestCompareVersion(t *testing.T) {
	casos := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0", "1.0.10", -1},
		{"1.0.0-dev", "1.0.0", -1},
		{"0.0.0-dev", "0.0.1", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-rc.1", "1.0.0-beta", 1},
		{"1.0.0+a", "1.0.0+b", 0},
	}
	for _, c := range casos {
		a, _ := ParseVersion(c.a)
		b, _ := ParseVersion(c.b)
		if got := a.Compare(b); got != c.want {
			t.Errorf("Compare(%s, %s) = %d, esperava %d", c.a, c.b, got, c.want)
		}
		if got := b.Compare(a); got != -c.want {
			t.Errorf("Compare(%s, %s) = %d, esperava %d (simetria)", c.b, c.a, got, -c.want)
		}
	}
}
