package api

// DailyLoggedMinutes soma os minutos lançados pelo usuário atual em cada dia
// do período [start, end] (datas YYYY-MM-DD). Lançamentos apagados ou de outros
// usuários são ignorados. Dias sem lançamento não aparecem no mapa.
func (t *TeamworkAPI) DailyLoggedMinutes(start, end string) (map[string]int, error) {
	entries, err := t.GetTimeEntriesForPeriodV2(start, end, false)
	if err != nil {
		return nil, err
	}

	minutos := make(map[string]int)
	for _, entry := range entries {
		if entry.DeletedAt != "" {
			continue
		}
		if entry.UserID != 0 && t.Config.UserID != 0 && entry.UserID != t.Config.UserID {
			continue
		}
		if entry.Date < start || entry.Date > end {
			continue
		}
		minutos[entry.Date] += entry.Minutes
	}
	return minutos, nil
}
