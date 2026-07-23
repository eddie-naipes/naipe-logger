package api

import (
	"strings"
	"time"
)

// teamworkDateLayouts cobre os formatos que a API do Teamwork devolve em campos
// de data conforme a versão do endpoint (v1/v2/v3).
var teamworkDateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02",
	"20060102",
}

// parseTeamworkDate interpreta uma data devolvida pela API, no fuso local, com
// a hora zerada. Devolve false quando o valor está ausente ou é irreconhecível
// — nesse caso o chamador deve omitir o registro em vez de inventar uma data.
func parseTeamworkDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	for _, layout := range teamworkDateLayouts {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local), true
	}

	return time.Time{}, false
}

// startOfToday devolve a meia-noite de hoje no fuso local.
func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}
