package api

import (
	"strings"
	"time"
)

// v2StartTime devolve a hora de início ("HH:MM") de um lançamento da API v2
// quando ele tem uma (hasStartTime). A hora é lida do mesmo valor de "date" e
// na mesma referência usada para o dia em v2EntryToReport, sem conversão de
// fuso, para que dia e hora nunca discordem. Sem hora, "".
func v2StartTime(entry v2TimeEntry) string {
	value := strings.TrimSpace(entry.Date)
	if !entry.HasStartTime || !strings.Contains(value, "T") {
		return ""
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("15:04")
		}
	}
	return ""
}
