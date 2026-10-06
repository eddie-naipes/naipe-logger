package reports

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"logTime-go/backend/api"
)

// CSV no formato que o Excel em pt-BR abre direto com duplo clique:
// separador ";", UTF-8 com BOM (sem ele os acentos viram lixo), números com
// vírgula decimal e datas dd/mm/aaaa.
const (
	csvSeparator = ';'
	csvBOM       = "\xEF\xBB\xBF"
	csvNewline   = "\r\n"
)

// CSVKind escolhe entre a lista de lançamentos e o resumo por tarefa.
type CSVKind string

const (
	CSVDetailed CSVKind = "detalhado"
	CSVSummary  CSVKind = "resumido"
)

// DetailedCSV gera uma linha por lançamento: data, projeto, tarefa,
// descrição, início, minutos, horas, cobrável.
func DetailedCSV(entries []api.TimeEntryReport) []byte {
	var b strings.Builder
	b.WriteString(csvBOM)
	writeRow(&b, "Data", "Projeto", "Tarefa", "Descrição", "Início", "Minutos", "Horas", "Cobrável")

	total := 0
	for _, e := range entries {
		total += e.Minutes
		writeRow(&b,
			formatDateBR(e.Date),
			textCell(nameOr(e.ProjectName, noProjectName)),
			textCell(nameOr(e.TaskName, noTaskName)),
			textCell(e.Description),
			textCell(e.StartTime),
			strconv.Itoa(e.Minutes),
			formatHours(e.Minutes),
			yesNo(e.IsBillable),
		)
	}
	writeRow(&b, "Total", "", "", "", "", strconv.Itoa(total), formatHours(total), "")
	return []byte(b.String())
}

// SummaryCSV gera uma linha por tarefa (com o projeto) e uma linha de total.
func SummaryCSV(r Report) []byte {
	var b strings.Builder
	b.WriteString(csvBOM)
	writeRow(&b, "Projeto", "Tarefa", "Lançamentos", "Minutos", "Horas", "Minutos cobráveis", "Horas cobráveis", "Cobrável")

	for _, t := range r.ByTask {
		writeRow(&b,
			textCell(t.ProjectName),
			textCell(t.TaskName),
			strconv.Itoa(t.EntryCount),
			strconv.Itoa(t.Minutes),
			formatHours(t.Minutes),
			strconv.Itoa(t.BillableMinutes),
			formatHours(t.BillableMinutes),
			billableLabel(t.BillableMinutes, t.Minutes),
		)
	}
	writeRow(&b, "Total", "", strconv.Itoa(r.EntryCount),
		strconv.Itoa(r.TotalMinutes), formatHours(r.TotalMinutes),
		strconv.Itoa(r.BillableMinutes), formatHours(r.BillableMinutes),
		billableLabel(r.BillableMinutes, r.TotalMinutes))
	return []byte(b.String())
}

// FileName monta o nome do CSV a partir de um período já validado.
func FileName(kind CSVKind, period Period) string {
	return fmt.Sprintf("horas_%s_%s_%s.csv", kind, period.StartDate(), period.EndDate())
}

func writeRow(b *strings.Builder, cells ...string) {
	for i, c := range cells {
		if i > 0 {
			b.WriteByte(csvSeparator)
		}
		b.WriteString(quote(c))
	}
	b.WriteString(csvNewline)
}

// quote aplica as regras do RFC 4180 com ";" como separador: campos com
// separador, aspas ou quebra de linha vão entre aspas, e aspas são dobradas.
func quote(cell string) string {
	if !strings.ContainsAny(cell, "\";\r\n") {
		return cell
	}
	return `"` + strings.ReplaceAll(cell, `"`, `""`) + `"`
}

// textCell neutraliza CSV injection: um texto vindo do Teamwork que comece
// com = + - @ (ou tab/CR, que alguns programas ignoram antes da fórmula)
// seria executado como fórmula pelo Excel. O apóstrofo faz a célula ser
// tratada como texto.
func textCell(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// formatHours devolve as horas com duas casas e vírgula decimal (1,50).
func formatHours(minutes int) string {
	return strings.Replace(strconv.FormatFloat(float64(minutes)/60, 'f', 2, 64), ".", ",", 1)
}

// formatDateBR converte AAAA-MM-DD em dd/mm/aaaa; valor irreconhecível vai
// como veio (e protegido contra injection).
func formatDateBR(date string) string {
	d, err := time.Parse(dateLayout, date)
	if err != nil {
		return textCell(date)
	}
	return d.Format("02/01/2006")
}

func yesNo(v bool) string {
	if v {
		return "Sim"
	}
	return "Não"
}

func billableLabel(billable, total int) string {
	switch {
	case billable == 0:
		return "Não"
	case billable >= total:
		return "Sim"
	default:
		return "Parcial"
	}
}
