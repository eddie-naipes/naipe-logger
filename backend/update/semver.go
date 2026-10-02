package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version é uma versão semântica simplificada (MAJOR.MINOR.PATCH[-pre][+build]).
// Metadados de build são ignorados na comparação, como manda o SemVer.
type Version struct {
	Major, Minor, Patch int
	Pre                 []string
}

// ParseVersion aceita "1.2.3", "v1.2.3", "1.2" (patch 0), "1.2.3-rc.1" e
// "1.2.3+build". Qualquer outra coisa é erro: comparar lixo poderia oferecer
// um "update" para uma versão mais velha.
func ParseVersion(s string) (Version, error) {
	raw := s
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, pre = s[:i], s[i+1:]
		if pre == "" {
			return Version{}, fmt.Errorf("versão inválida %q", raw)
		}
	}

	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return Version{}, fmt.Errorf("versão inválida %q", raw)
	}
	nums := [3]int{}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || p == "" {
			return Version{}, fmt.Errorf("versão inválida %q", raw)
		}
		nums[i] = n
	}

	v := Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}
	if pre != "" {
		v.Pre = strings.Split(pre, ".")
		for _, id := range v.Pre {
			if id == "" {
				return Version{}, fmt.Errorf("versão inválida %q", raw)
			}
		}
	}
	return v, nil
}

// IsPrerelease informa se a versão tem sufixo (ex.: "-dev", "-rc.1").
func (v Version) IsPrerelease() bool { return len(v.Pre) > 0 }

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	return s
}

// Compare devolve -1, 0 ou 1 conforme v seja menor, igual ou maior que o.
// Pré-releases ficam abaixo da versão final correspondente (1.0.0-dev < 1.0.0).
func (v Version) Compare(o Version) int {
	for _, d := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Patch, o.Patch}} {
		if d[0] != d[1] {
			return cmpInt(d[0], d[1])
		}
	}
	switch {
	case len(v.Pre) == 0 && len(o.Pre) == 0:
		return 0
	case len(v.Pre) == 0:
		return 1
	case len(o.Pre) == 0:
		return -1
	}
	for i := 0; i < len(v.Pre) && i < len(o.Pre); i++ {
		if c := comparePreID(v.Pre[i], o.Pre[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(v.Pre), len(o.Pre))
}

// comparePreID segue o SemVer: identificadores numéricos comparam como
// números e ficam abaixo dos alfanuméricos.
func comparePreID(a, b string) int {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		return cmpInt(na, nb)
	case errA == nil:
		return -1
	case errB == nil:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
