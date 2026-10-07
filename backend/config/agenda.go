package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Importação de reuniões da agenda (backend/agenda). Aqui ficam só os metadados
// das agendas e as regras de mapeamento: os links iCal são segredos e vivem no
// cofre do sistema (security.StoreAgendaURL), nunca em config.json.

// Arredondamentos aceitos em AgendaSettings.Rounding.
const (
	AgendaRoundingExact = "exact"
	AgendaRounding15    = "15"
)

// AgendaCalendar descreve uma agenda cadastrada, sem o link.
type AgendaCalendar struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// MaskedURL é host + "…" + últimos caracteres do link, para exibição.
	MaskedURL string `json:"maskedUrl"`
	AddedAt   string `json:"addedAt"`
}

// AgendaTask identifica a tarefa do Teamwork de uma regra.
type AgendaTask struct {
	TaskID      int    `json:"taskId"`
	TaskName    string `json:"taskName"`
	ProjectID   int    `json:"projectId"`
	ProjectName string `json:"projectName"`
}

// AgendaRule associa eventos a uma tarefa pelo título; a primeira que casa vale.
type AgendaRule struct {
	// Match é uma palavra-chave (sem diferenciar maiúsculas) ou, com IsRegex,
	// uma expressão regular.
	Match   string     `json:"match"`
	IsRegex bool       `json:"isRegex"`
	Task    AgendaTask `json:"task"`
	// Description substitui o título do evento ("" usa o título).
	Description string `json:"description"`
}

// AgendaSettings é a parte de config.json da importação da agenda.
type AgendaSettings struct {
	// Calendars é mantido pelos bindings de adicionar/remover agenda;
	// SetAgendaSettings o ignora.
	Calendars []AgendaCalendar `json:"calendars"`
	Rules     []AgendaRule     `json:"rules"`
	// DefaultTask recebe eventos sem regra (TaskID 0 = deixar para escolher).
	DefaultTask AgendaTask `json:"defaultTask"`
	IgnoreWords []string   `json:"ignoreWords"`
	// MinMinutes descarta eventos mais curtos.
	MinMinutes int `json:"minMinutes"`
	// Rounding: "exact" ou "15".
	Rounding string `json:"rounding"`
	// UserEmail identifica o usuário nos convites para ignorar os recusados.
	UserEmail          string `json:"userEmail"`
	Billable           bool   `json:"billable"`
	IncludeTransparent bool   `json:"includeTransparent"`
}

// defaultAgendaSettings vale para config.json antigos, sem o campo: duração
// exata, descarta eventos de menos de 5 min e lança como faturável (o mesmo
// padrão das entradas novas em Tarefas).
func defaultAgendaSettings() AgendaSettings {
	return AgendaSettings{Rounding: AgendaRoundingExact, MinMinutes: 5, Billable: true}
}

// Normalized devolve uma cópia sem slices nil e com valores aparados.
func (s AgendaSettings) Normalized() AgendaSettings {
	out := s
	out.Calendars = append([]AgendaCalendar{}, s.Calendars...)
	out.Rules = make([]AgendaRule, 0, len(s.Rules))
	for _, r := range s.Rules {
		r.Match = strings.TrimSpace(r.Match)
		r.Description = strings.TrimSpace(r.Description)
		out.Rules = append(out.Rules, r)
	}
	out.IgnoreWords = make([]string, 0, len(s.IgnoreWords))
	for _, w := range s.IgnoreWords {
		if w = strings.TrimSpace(w); w != "" {
			out.IgnoreWords = append(out.IgnoreWords, w)
		}
	}
	out.UserEmail = strings.TrimSpace(s.UserEmail)
	if out.Rounding != AgendaRounding15 {
		out.Rounding = AgendaRoundingExact
	}
	return out
}

// Validate confere regras e campos numéricos.
func (s AgendaSettings) Validate() error {
	for i, r := range s.Rules {
		if r.Match == "" {
			return fmt.Errorf("regra %d: informe a palavra-chave ou a expressão", i+1)
		}
		if r.IsRegex {
			if _, err := regexp.Compile(r.Match); err != nil {
				return fmt.Errorf("regra %d: expressão regular inválida: %v", i+1, err)
			}
		}
		if r.Task.TaskID <= 0 {
			return fmt.Errorf("regra %d: escolha a tarefa", i+1)
		}
	}
	if s.MinMinutes < 0 || s.MinMinutes > 24*60 {
		return fmt.Errorf("duração mínima inválida: %d", s.MinMinutes)
	}
	if s.UserEmail != "" && (!strings.Contains(s.UserEmail, "@") || strings.ContainsAny(s.UserEmail, " <>")) {
		return fmt.Errorf("e-mail inválido: %q", s.UserEmail)
	}
	return nil
}

// GetAgendaSettings devolve uma cópia da configuração da agenda.
func (m *Manager) GetAgendaSettings() AgendaSettings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.Agenda.Normalized()
}

// SetAgendaSettings valida e grava regras e preferências, preservando a lista
// de agendas atual.
func (m *Manager) SetAgendaSettings(s AgendaSettings) error {
	s = s.Normalized()
	if err := s.Validate(); err != nil {
		return err
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	s.Calendars = append([]AgendaCalendar{}, m.appConfig.Agenda.Calendars...)
	m.appConfig.Agenda = s
	return m.saveLocked()
}

// AddAgendaCalendar acrescenta uma agenda (sem o link).
func (m *Manager) AddAgendaCalendar(c AgendaCalendar) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, existente := range m.appConfig.Agenda.Calendars {
		if existente.ID == c.ID {
			return fmt.Errorf("agenda já cadastrada: %s", c.ID)
		}
	}
	m.appConfig.Agenda.Calendars = append(m.appConfig.Agenda.Calendars, c)
	return m.saveLocked()
}

// RemoveAgendaCalendar tira a agenda da lista. Uma agenda inexistente não é erro.
func (m *Manager) RemoveAgendaCalendar(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	lista := m.appConfig.Agenda.Calendars[:0:0]
	for _, c := range m.appConfig.Agenda.Calendars {
		if c.ID != id {
			lista = append(lista, c)
		}
	}
	m.appConfig.Agenda.Calendars = lista
	return m.saveLocked()
}
