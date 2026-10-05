package reports

import (
	"strings"
	"testing"

	"logTime-go/backend/api"
)

func linhas(t *testing.T, data []byte) []string {
	t.Helper()
	s := string(data)
	// O BOM UTF-8 são exatamente os bytes EF BB BF no início do arquivo.
	if !strings.HasPrefix(s, string([]byte{0xEF, 0xBB, 0xBF})) {
		t.Fatal("CSV sem BOM UTF-8: o Excel mostraria os acentos errados")
	}
	s = strings.TrimPrefix(s, csvBOM)
	if !strings.HasSuffix(s, csvNewline) {
		t.Error("CSV deveria terminar com CRLF")
	}
	return strings.Split(strings.TrimSuffix(s, csvNewline), csvNewline)
}

func TestDetailedCSVFormatoExcelPtBR(t *testing.T) {
	data := DetailedCSV([]api.TimeEntryReport{
		{Date: "2026-06-01", ProjectName: "Alfa", TaskName: "Dev", Description: "Ajuste", StartTime: "09:00", Minutes: 90, IsBillable: true},
		{Date: "2026-06-02", ProjectName: "Beta", TaskName: "Reunião", Minutes: 45},
	})
	l := linhas(t, data)

	if l[0] != "Data;Projeto;Tarefa;Descrição;Início;Minutos;Horas;Cobrável" {
		t.Errorf("cabeçalho = %q", l[0])
	}
	if l[1] != "01/06/2026;Alfa;Dev;Ajuste;09:00;90;1,50;Sim" {
		t.Errorf("linha 1 = %q", l[1])
	}
	if l[2] != "02/06/2026;Beta;Reunião;;;45;0,75;Não" {
		t.Errorf("linha 2 = %q", l[2])
	}
	if l[3] != "Total;;;;;135;2,25;" {
		t.Errorf("total = %q", l[3])
	}
}

func TestDetailedCSVEscapaAspasSeparadorEQuebra(t *testing.T) {
	data := DetailedCSV([]api.TimeEntryReport{
		{Date: "2026-06-01", ProjectName: `Cliente "VIP"`, TaskName: "a;b", Description: "linha1\nlinha2", Minutes: 60},
	})
	s := string(data)
	for _, want := range []string{`"Cliente ""VIP"""`, `"a;b"`, "\"linha1\nlinha2\""} {
		if !strings.Contains(s, want) {
			t.Errorf("faltou %q em %q", want, s)
		}
	}
}

func TestDetailedCSVNeutralizaInjection(t *testing.T) {
	data := DetailedCSV([]api.TimeEntryReport{
		{Date: "2026-06-01", ProjectName: "=HYPERLINK(\"x\")", TaskName: "+SOMA(A1)", Description: "-2+3", StartTime: "@cmd", Minutes: 60},
	})
	l := linhas(t, data)
	campos := l[1]
	for _, want := range []string{`"'=HYPERLINK(""x"")"`, "'+SOMA(A1)", "'-2+3", "'@cmd"} {
		if !strings.Contains(campos, want) {
			t.Errorf("faltou %q em %q", want, campos)
		}
	}
	// Números do próprio relatório não levam apóstrofo.
	if !strings.HasSuffix(campos, ";60;1,00;Não") {
		t.Errorf("colunas numéricas alteradas: %q", campos)
	}
}

func TestSummaryCSV(t *testing.T) {
	r := Build(lancamentos(), periodo(t, "2026-06-01", "2026-06-07"), uteis, 480, 42)
	l := linhas(t, SummaryCSV(r))

	if l[0] != "Projeto;Tarefa;Lançamentos;Minutos;Horas;Minutos cobráveis;Horas cobráveis;Cobrável" {
		t.Errorf("cabeçalho = %q", l[0])
	}
	if l[1] != "Beta;Suporte;1;480;8,00;480;8,00;Sim" {
		t.Errorf("linha 1 = %q", l[1])
	}
	if len(l) != 5 || l[4] != "Total;;4;960;16,00;780;13,00;Parcial" {
		t.Errorf("linhas = %q", l)
	}
}

func TestFormatHoursComVirgula(t *testing.T) {
	cases := map[int]string{0: "0,00", 1: "0,02", 20: "0,33", 480: "8,00", 125: "2,08"}
	for min, want := range cases {
		if got := formatHours(min); got != want {
			t.Errorf("formatHours(%d) = %q, esperava %q", min, got, want)
		}
	}
}
