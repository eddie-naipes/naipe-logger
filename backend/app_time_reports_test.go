package backend

import (
	"errors"
	"testing"
)

func TestRelatoriosExigemConexao(t *testing.T) {
	a := appSemConexao()

	if _, err := a.GetTimeReportSummary("2026-01-01", "2026-01-31"); !errors.Is(err, errAPINaoConfigurada) {
		t.Errorf("GetTimeReportSummary sem conexão = %v", err)
	}
	if _, err := a.ExportTimeReportCSV("2026-01-01", "2026-01-31", true); !errors.Is(err, errAPINaoConfigurada) {
		t.Errorf("ExportTimeReportCSV sem conexão = %v", err)
	}
}

// As datas entram no nome do arquivo: são recusadas antes de qualquer acesso
// à rede ou ao disco.
func TestExportTimeReportCSVRecusaDatasInvalidas(t *testing.T) {
	a := appSemConexao()
	for _, c := range [][2]string{{"../../x", "2026-01-31"}, {"2026-02-01", "2026-01-01"}, {"2026-01-01", "2026-1-31"}} {
		_, err := a.ExportTimeReportCSV(c[0], c[1], false)
		if err == nil || errors.Is(err, errAPINaoConfigurada) {
			t.Errorf("ExportTimeReportCSV(%q, %q) = %v, esperava erro de data", c[0], c[1], err)
		}
	}
}

func TestCalendarioSemProvedorNaoQuebra(t *testing.T) {
	a := appSemConexao()
	dias, err := a.GetExtraNonWorkingDays(2026)
	if err != nil || dias == nil || len(dias) != 0 {
		t.Errorf("GetExtraNonWorkingDays = %v, %v", dias, err)
	}
	if _, err := a.GetExtraNonWorkingDays(1); err == nil {
		t.Error("ano absurdo deveria ser recusado")
	}
	if len(a.GetBrazilianStates()) != 27 {
		t.Error("deveria listar as 27 UFs")
	}
}
