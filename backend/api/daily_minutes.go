package api

// DailyLoggedMinutes devolve os minutos lançados pelo usuário atual em cada dia
// (chave 'YYYY-MM-DD') do período [start, end]. Dias sem lançamento não
// aparecem no mapa. Usado pelo "completar período" e pelos lembretes.
//
// Usa a listagem v2 sem as entradas apagadas e ainda descarta, por segurança,
// lançamentos de outros usuários e fora do período: o filtro da URL é o que
// garante isso hoje, mas um total de jornada errado faria o "completar
// período" lançar a menos (ou o lembrete calar) sem ninguém perceber.
func (t *TeamworkAPI) DailyLoggedMinutes(start, end string) (map[string]int, error) {
	entries, err := t.GetTimeEntriesForPeriodV2(start, end, false)
	if err != nil {
		return nil, err
	}

	porDia := make(map[string]int)
	for _, entry := range entries {
		if entry.DeletedAt != "" {
			continue
		}
		// UserID 0 = a resposta não informou o dono; o filtro da URL já
		// restringiu ao usuário atual, então a entrada é dele.
		if entry.UserID != 0 && t.Config.UserID != 0 && entry.UserID != t.Config.UserID {
			continue
		}
		if entry.Date == "" || entry.Date < start || entry.Date > end || entry.Minutes <= 0 {
			continue
		}
		porDia[entry.Date] += entry.Minutes
	}
	return porDia, nil
}
