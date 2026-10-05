// Package holidays monta o calendário de dias não úteis além dos feriados
// nacionais: feriados estaduais de data fixa (tabela embutida), feriados
// municipais/pontes cadastrados pelo usuário e períodos de férias.
package holidays

import "sort"

// StateHoliday é um feriado estadual de data fixa, válido todo ano.
type StateHoliday struct {
	// MonthDay é "MM-DD".
	MonthDay string `json:"monthDay"`
	Name     string `json:"name"`
	// Source descreve a origem (lei/constituição estadual).
	Source string `json:"source"`
}

// State é uma unidade da federação e seus feriados estaduais de data fixa.
type State struct {
	UF       string         `json:"uf"`
	Name     string         `json:"name"`
	Holidays []StateHoliday `json:"holidays"`
}

// stateTable lista só feriados estaduais de DATA FIXA amplamente conhecidos e
// previstos na legislação do estado. A lista é deliberadamente conservadora:
//   - feriados nacionais (Tiradentes, Consciência Negra desde a Lei
//     14.759/2023 etc.) não são repetidos aqui — a BrasilAPI já os traz;
//   - feriados móveis (ex.: Nossa Senhora da Penha no ES) e datas
//     transferidas para domingo (ex.: Data Magna de SC) ficam de fora;
//   - estados sem feriado estadual fixo consolidado ficam com a lista vazia.
//
// O usuário pode desligar qualquer item (DisabledStateHolidays) e cadastrar
// o que faltar como feriado personalizado — a UI deixa isso explícito.
var stateTable = []State{
	{UF: "AC", Name: "Acre", Holidays: []StateHoliday{
		{MonthDay: "06-15", Name: "Aniversário do Estado do Acre", Source: "Legislação estadual do Acre"},
		{MonthDay: "08-06", Name: "Início da Revolução Acreana", Source: "Legislação estadual do Acre"},
	}},
	{UF: "AL", Name: "Alagoas", Holidays: []StateHoliday{
		{MonthDay: "06-24", Name: "São João", Source: "Legislação estadual de Alagoas"},
		{MonthDay: "06-29", Name: "São Pedro", Source: "Legislação estadual de Alagoas"},
		{MonthDay: "09-16", Name: "Emancipação Política de Alagoas", Source: "Legislação estadual de Alagoas"},
	}},
	{UF: "AM", Name: "Amazonas", Holidays: []StateHoliday{
		{MonthDay: "09-05", Name: "Elevação do Amazonas à categoria de Província", Source: "Legislação estadual do Amazonas"},
	}},
	{UF: "AP", Name: "Amapá", Holidays: []StateHoliday{
		{MonthDay: "03-19", Name: "Dia de São José", Source: "Legislação estadual do Amapá"},
		{MonthDay: "09-13", Name: "Criação do Território Federal do Amapá", Source: "Legislação estadual do Amapá"},
	}},
	{UF: "BA", Name: "Bahia", Holidays: []StateHoliday{
		{MonthDay: "07-02", Name: "Independência da Bahia", Source: "Constituição do Estado da Bahia"},
	}},
	{UF: "CE", Name: "Ceará", Holidays: []StateHoliday{
		{MonthDay: "03-19", Name: "Dia de São José", Source: "Legislação estadual do Ceará"},
		{MonthDay: "03-25", Name: "Data Magna do Ceará (Abolição da escravatura)", Source: "Legislação estadual do Ceará"},
	}},
	{UF: "DF", Name: "Distrito Federal", Holidays: []StateHoliday{
		{MonthDay: "11-30", Name: "Dia do Evangélico", Source: "Legislação distrital"},
	}},
	{UF: "ES", Name: "Espírito Santo"},
	{UF: "GO", Name: "Goiás"},
	{UF: "MA", Name: "Maranhão", Holidays: []StateHoliday{
		{MonthDay: "07-28", Name: "Adesão do Maranhão à Independência", Source: "Legislação estadual do Maranhão"},
	}},
	{UF: "MG", Name: "Minas Gerais"},
	{UF: "MS", Name: "Mato Grosso do Sul", Holidays: []StateHoliday{
		{MonthDay: "10-11", Name: "Criação do Estado de Mato Grosso do Sul", Source: "Legislação estadual de Mato Grosso do Sul"},
	}},
	{UF: "MT", Name: "Mato Grosso"},
	{UF: "PA", Name: "Pará", Holidays: []StateHoliday{
		{MonthDay: "08-15", Name: "Adesão do Pará à Independência", Source: "Legislação estadual do Pará"},
	}},
	{UF: "PB", Name: "Paraíba", Holidays: []StateHoliday{
		{MonthDay: "08-05", Name: "Fundação do Estado da Paraíba", Source: "Legislação estadual da Paraíba"},
	}},
	{UF: "PE", Name: "Pernambuco", Holidays: []StateHoliday{
		{MonthDay: "03-06", Name: "Data Magna de Pernambuco (Revolução Pernambucana)", Source: "Legislação estadual de Pernambuco"},
	}},
	{UF: "PI", Name: "Piauí", Holidays: []StateHoliday{
		{MonthDay: "10-19", Name: "Dia do Piauí", Source: "Legislação estadual do Piauí"},
	}},
	{UF: "PR", Name: "Paraná", Holidays: []StateHoliday{
		{MonthDay: "12-19", Name: "Emancipação Política do Paraná", Source: "Legislação estadual do Paraná"},
	}},
	{UF: "RJ", Name: "Rio de Janeiro", Holidays: []StateHoliday{
		{MonthDay: "04-23", Name: "Dia de São Jorge", Source: "Lei estadual RJ 5.198/2008"},
	}},
	{UF: "RN", Name: "Rio Grande do Norte", Holidays: []StateHoliday{
		{MonthDay: "10-03", Name: "Mártires de Cunhaú e Uruaçu", Source: "Legislação estadual do Rio Grande do Norte"},
	}},
	{UF: "RO", Name: "Rondônia", Holidays: []StateHoliday{
		{MonthDay: "01-04", Name: "Criação do Estado de Rondônia", Source: "Legislação estadual de Rondônia"},
		{MonthDay: "06-18", Name: "Dia do Evangélico", Source: "Legislação estadual de Rondônia"},
	}},
	{UF: "RR", Name: "Roraima", Holidays: []StateHoliday{
		{MonthDay: "10-05", Name: "Criação do Estado de Roraima", Source: "Legislação estadual de Roraima"},
	}},
	{UF: "RS", Name: "Rio Grande do Sul", Holidays: []StateHoliday{
		{MonthDay: "09-20", Name: "Revolução Farroupilha", Source: "Legislação estadual do Rio Grande do Sul"},
	}},
	{UF: "SC", Name: "Santa Catarina"},
	{UF: "SE", Name: "Sergipe", Holidays: []StateHoliday{
		{MonthDay: "07-08", Name: "Emancipação Política de Sergipe", Source: "Legislação estadual de Sergipe"},
	}},
	{UF: "SP", Name: "São Paulo", Holidays: []StateHoliday{
		{MonthDay: "07-09", Name: "Revolução Constitucionalista de 1932", Source: "Lei estadual SP 9.497/1997"},
	}},
	{UF: "TO", Name: "Tocantins", Holidays: []StateHoliday{
		{MonthDay: "10-05", Name: "Criação do Estado do Tocantins", Source: "Legislação estadual do Tocantins"},
	}},
}

// States devolve todas as UFs (ordenadas pela sigla) com seus feriados
// estaduais. É uma cópia: quem chama pode alterá-la.
func States() []State {
	out := make([]State, 0, len(stateTable))
	for _, s := range stateTable {
		s.Holidays = append([]StateHoliday{}, s.Holidays...)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UF < out[j].UF })
	return out
}

// StateByUF devolve a UF (sigla em maiúsculas) e se ela existe.
func StateByUF(uf string) (State, bool) {
	for _, s := range stateTable {
		if s.UF == uf {
			s.Holidays = append([]StateHoliday{}, s.Holidays...)
			return s, true
		}
	}
	return State{}, false
}
