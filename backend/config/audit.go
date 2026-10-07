package config

import (
	"fmt"
	"regexp"
	"strings"
)

// AuditSettings configura o "fechamento do mês" (pacote backend/audit, seção
// "audit" de config.json). Como Load decodifica sobre defaultAppConfig, um
// config.json antigo (sem a seção ou com campos faltando) fica com os padrões.
type AuditSettings struct {
	// DailyLimitMinutes é o limite diário acima do qual o dia vira aviso;
	// 0 desliga a verificação. Padrão 600 (10h).
	DailyLimitMinutes int `json:"dailyLimitMinutes"`
	// GenericDescriptions são descrições vagas demais ("reunião", "ajustes").
	// Padrão vazio.
	GenericDescriptions []string `json:"genericDescriptions"`
	// IgnoredIssues são as chaves (audit.Issue.Key) dos problemas que o
	// usuário mandou ignorar. Só mudam por IgnoreAuditIssue/UnignoreAuditIssue:
	// SetAuditSettings preserva a lista gravada.
	IgnoredIssues []string `json:"ignoredIssues"`
}

const (
	// DefaultAuditDailyLimitMinutes é o limite diário padrão (10h).
	DefaultAuditDailyLimitMinutes = 10 * 60
	maxAuditDailyLimitMinutes     = 24 * 60
	maxGenericDescriptions        = 100
	maxGenericDescriptionLen      = 200
	// maxIgnoredIssues limita a lista; ao estourar, as mais antigas saem.
	maxIgnoredIssues = 2000
)

// auditIssueKeyPattern aceita só o formato das chaves geradas pelo
// backend/audit ("tipo:AAAA-MM-DD" ou "tipo:id1,id2").
var auditIssueKeyPattern = regexp.MustCompile(`^[a-z_]{1,40}:[0-9][0-9,\-]{0,180}$`)

// DefaultAuditSettings devolve os padrões da auditoria.
func DefaultAuditSettings() AuditSettings {
	return AuditSettings{
		DailyLimitMinutes:   DefaultAuditDailyLimitMinutes,
		GenericDescriptions: []string{},
		IgnoredIssues:       []string{},
	}
}

// normalized corrige valores inválidos vindos de um config.json editado à
// mão e garante slices não nulos (o frontend recebe [] e não null).
func (s AuditSettings) normalized() AuditSettings {
	if s.DailyLimitMinutes < 0 || s.DailyLimitMinutes > maxAuditDailyLimitMinutes {
		s.DailyLimitMinutes = DefaultAuditDailyLimitMinutes
	}
	s.GenericDescriptions = normalizeGenericDescriptions(s.GenericDescriptions)

	ignored := make([]string, 0, len(s.IgnoredIssues))
	seen := map[string]bool{}
	for _, k := range s.IgnoredIssues {
		if !auditIssueKeyPattern.MatchString(k) || seen[k] {
			continue
		}
		seen[k] = true
		ignored = append(ignored, k)
	}
	s.IgnoredIssues = ignored
	return s
}

// normalizeGenericDescriptions apara, descarta vazias e repetidas (sem
// diferenciar maiúsculas) e mantém a ordem.
func normalizeGenericDescriptions(list []string) []string {
	out := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, d := range list {
		d = strings.Join(strings.Fields(d), " ")
		key := strings.ToLower(d)
		if d == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	return out
}

// GetAuditSettings devolve uma cópia da configuração da auditoria.
func (m *Manager) GetAuditSettings() AuditSettings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.Audit.normalized()
}

// SetAuditSettings valida e grava limite diário e descrições genéricas. A
// lista de ignorados recebida é descartada: vale a gravada.
func (m *Manager) SetAuditSettings(settings AuditSettings) error {
	if settings.DailyLimitMinutes < 0 || settings.DailyLimitMinutes > maxAuditDailyLimitMinutes {
		return fmt.Errorf("limite diário inválido: %d minutos (use de 0 a 24h; 0 desliga)", settings.DailyLimitMinutes)
	}
	generic := normalizeGenericDescriptions(settings.GenericDescriptions)
	if len(generic) > maxGenericDescriptions {
		return fmt.Errorf("descrições genéricas demais: %d (máximo %d)", len(generic), maxGenericDescriptions)
	}
	for _, d := range generic {
		if len([]rune(d)) > maxGenericDescriptionLen {
			return fmt.Errorf("descrição genérica longa demais (máximo %d caracteres)", maxGenericDescriptionLen)
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	current := m.appConfig.Audit.normalized()
	current.DailyLimitMinutes = settings.DailyLimitMinutes
	current.GenericDescriptions = generic
	m.appConfig.Audit = current
	return m.saveLocked()
}

// IgnoreAuditIssue grava a chave de um problema como ignorado (idempotente).
func (m *Manager) IgnoreAuditIssue(key string) error {
	if !auditIssueKeyPattern.MatchString(key) {
		return fmt.Errorf("chave de problema inválida: %q", key)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	current := m.appConfig.Audit.normalized()
	for _, k := range current.IgnoredIssues {
		if k == key {
			return nil
		}
	}
	current.IgnoredIssues = append(current.IgnoredIssues, key)
	if extra := len(current.IgnoredIssues) - maxIgnoredIssues; extra > 0 {
		current.IgnoredIssues = current.IgnoredIssues[extra:]
	}
	m.appConfig.Audit = current
	return m.saveLocked()
}

// UnignoreAuditIssue volta a exibir um problema ignorado (idempotente).
func (m *Manager) UnignoreAuditIssue(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	current := m.appConfig.Audit.normalized()
	kept := make([]string, 0, len(current.IgnoredIssues))
	for _, k := range current.IgnoredIssues {
		if k != key {
			kept = append(kept, k)
		}
	}
	if len(kept) == len(current.IgnoredIssues) {
		return nil
	}
	current.IgnoredIssues = kept
	m.appConfig.Audit = current
	return m.saveLocked()
}
