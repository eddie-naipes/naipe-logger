package holidays

import (
	"context"
	"testing"
	"time"

	"logTime-go/backend/api"
)

func TestTabelaTemAs27UFsSemRepeticao(t *testing.T) {
	states := States()
	if len(states) != 27 {
		t.Fatalf("esperava 27 UFs, veio %d", len(states))
	}
	vistas := map[string]bool{}
	for _, s := range states {
		if len(s.UF) != 2 || s.Name == "" {
			t.Errorf("UF malformada: %+v", s)
		}
		if vistas[s.UF] {
			t.Errorf("UF repetida: %s", s.UF)
		}
		vistas[s.UF] = true
	}
}

func TestFeriadosEstaduaisTemDataValidaEFonte(t *testing.T) {
	for _, s := range States() {
		dias := map[string]bool{}
		for _, h := range s.Holidays {
			// 2024 é bissexto: aceita qualquer MM-DD real.
			if _, err := time.Parse("2006-01-02", "2024-"+h.MonthDay); err != nil {
				t.Errorf("%s: data inválida %q", s.UF, h.MonthDay)
			}
			if h.Name == "" || h.Source == "" {
				t.Errorf("%s %s: sem nome ou fonte", s.UF, h.MonthDay)
			}
			if dias[h.MonthDay] {
				t.Errorf("%s: data repetida %s", s.UF, h.MonthDay)
			}
			dias[h.MonthDay] = true
		}
	}
}

// Nacionais já vêm da BrasilAPI/fallback: a tabela estadual não pode
// repeti-los (ex.: 21/04 em MG/DF, 20/11 desde 2024).
func TestFeriadosEstaduaisNaoDuplicamNacionais(t *testing.T) {
	for year := 2024; year <= 2035; year++ {
		nacionais, err := (&api.FixedHolidaysProvider{}).GetHolidays(context.Background(), year)
		if err != nil {
			t.Fatal(err)
		}
		datas := map[string]string{}
		for _, h := range nacionais {
			datas[h.Date[5:]] = h.Name
		}
		for _, s := range States() {
			for _, h := range s.Holidays {
				if nome, dup := datas[h.MonthDay]; dup && !mobileOnly(nome) {
					t.Errorf("%s %s (%s) duplica o nacional %q", s.UF, h.MonthDay, h.Name, nome)
				}
			}
		}
	}
}

// mobileOnly: coincidir num ano com um feriado móvel (Carnaval, Páscoa...)
// não é duplicação; só os nacionais de data fixa contam.
func mobileOnly(nome string) bool {
	switch nome {
	case "Carnaval", "Carnaval (segunda-feira)", "Sexta-feira Santa", "Páscoa", "Corpus Christi":
		return true
	}
	return false
}

func TestEstadosConhecidosTemOsFeriadosEsperados(t *testing.T) {
	cases := map[string]string{"SP": "07-09", "RJ": "04-23", "BA": "07-02", "PE": "03-06", "RS": "09-20"}
	for uf, md := range cases {
		s, ok := StateByUF(uf)
		if !ok || !stateHasMonthDay(s, md) {
			t.Errorf("%s deveria ter feriado em %s", uf, md)
		}
	}
	if s, _ := StateByUF("MG"); len(s.Holidays) != 0 {
		t.Errorf("MG não tem feriado estadual fixo além de 21/04 (nacional): %+v", s.Holidays)
	}
}

func TestStatesDevolveCopia(t *testing.T) {
	s := States()
	s[0].Holidays = append(s[0].Holidays, StateHoliday{MonthDay: "01-01"})
	original, _ := StateByUF(s[0].UF)
	if len(original.Holidays) == len(s[0].Holidays) {
		t.Error("States deveria devolver cópia da tabela")
	}
}
