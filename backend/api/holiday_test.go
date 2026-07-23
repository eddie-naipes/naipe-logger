package api

import "testing"

// Datas oficiais da Páscoa (domingo) — referência para o algoritmo de Gauss,
// que alimenta Carnaval, Sexta-feira Santa e Corpus Christi.
func TestCalculateEaster(t *testing.T) {
	cases := map[int]string{
		2020: "2020-04-12",
		2021: "2021-04-04",
		2022: "2022-04-17",
		2023: "2023-04-09",
		2024: "2024-03-31",
		2025: "2025-04-20",
		2026: "2026-04-05",
		2027: "2027-03-28",
		2030: "2030-04-21",
		2038: "2038-04-25", // limite superior do intervalo comum
	}

	for year, want := range cases {
		got := calculateEaster(year).Format("2006-01-02")
		if got != want {
			t.Errorf("calculateEaster(%d) = %s, esperava %s", year, got, want)
		}
	}
}

func TestCalculateMobileHolidays(t *testing.T) {
	// 2025: Páscoa em 20/04. Carnaval 04/03, Sexta-feira Santa 18/04,
	// Corpus Christi 19/06.
	want := map[string]string{
		"Carnaval":          "2025-03-04",
		"Sexta-feira Santa": "2025-04-18",
		"Páscoa":            "2025-04-20",
		"Corpus Christi":    "2025-06-19",
	}

	holidays := calculateMobileHolidays(2025)
	if len(holidays) != len(want) {
		t.Fatalf("calculateMobileHolidays devolveu %d feriados, esperava %d", len(holidays), len(want))
	}

	for _, h := range holidays {
		expected, ok := want[h.Name]
		if !ok {
			t.Errorf("feriado inesperado: %s", h.Name)
			continue
		}
		if h.Date != expected {
			t.Errorf("%s = %s, esperava %s", h.Name, h.Date, expected)
		}
	}
}

func TestConvertBrasilAPIDate(t *testing.T) {
	cases := []struct {
		in   string
		year int
		want string
	}{
		{"25/12", 2025, "2025-12-25"},
		{"01/01", 2025, "2025-01-01"},
		{"07/09/2026", 2025, "2026-09-07"}, // ano explícito prevalece
		{"", 2025, ""},
		{"lixo", 2025, ""},
	}

	for _, tc := range cases {
		if got := convertBrasilAPIDate(tc.in, tc.year); got != tc.want {
			t.Errorf("convertBrasilAPIDate(%q, %d) = %q, esperava %q", tc.in, tc.year, got, tc.want)
		}
	}
}
