package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// anonymizer troca dados pessoais de respostas da API por valores fictícios,
// preservando estrutura, tipos (texto continua texto, número continua número),
// datas, minutos e metadados de paginação.
//
// A regra é "nega por padrão": todo texto é substituído, exceto datas,
// números escritos como texto e valores de chaves enumeradas conhecidas
// (status, type, ...). IDs (chaves "id", "*Id", "*Ids" e chaves numéricas de
// mapas como included.projects["123"]) são remapeados para valores fictícios
// consistentes entre todos os arquivos de uma mesma execução, para que as
// referências cruzadas (tarefa -> lista -> projeto, responsável = usuário)
// continuem resolvendo. O resultado é determinístico: as chaves são
// percorridas em ordem alfabética.
type anonymizer struct {
	ids     map[string]string
	nextID  int64
	strs    map[string]string
	counter map[string]int
	// kept registra os textos mantidos por chave, para revisão manual.
	kept map[string]map[string]bool
}

func newAnonymizer() *anonymizer {
	return &anonymizer{
		ids:     map[string]string{},
		nextID:  1000,
		strs:    map[string]string{},
		counter: map[string]int{},
		kept:    map[string]map[string]bool{},
	}
}

// enumKeys são chaves cujo valor é um código de domínio, não texto livre.
var enumKeys = map[string]bool{
	"type": true, "status": true, "priority": true, "state": true,
	"substatus": true, "projectstatus": true, "billabletype": true,
	"invoicedtype": true, "kind": true, "entitytype": true, "ownertype": true,
	"usertype": true, "progress": true, "sortorder": true, "sortby": true,
	"color": true, "timezone": true, "timeformat": true, "dateformat": true,
	"language": true, "languagecode": true, "currency": true, "currencycode": true,
	"subtype": true,
}

var (
	reNumeric = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	reDigits  = regexp.MustCompile(`^\d+$`)
	reHexCor  = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
	reHora    = regexp.MustCompile(`^\d{1,2}:\d{2}(:\d{2})?$`)
)

var dateLayouts = []string{
	time.RFC3339, time.RFC3339Nano, "2006-01-02", "20060102",
	"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02T15:04:05.000Z",
	"20060102T150405Z",
}

func looksLikeDate(s string) bool {
	if reHora.MatchString(s) {
		return true
	}
	for _, layout := range dateLayouts {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

// isIDKey diz se a chave guarda um identificador numérico: id, *Id, *Ids e
// *By (createdBy, updatedBy, loggedBy guardam o ID de uma pessoa).
func isIDKey(key string) bool {
	k := strings.ToLower(key)
	return k == "id" || k == "ids" || strings.HasSuffix(k, "id") ||
		strings.HasSuffix(k, "ids") || strings.HasSuffix(k, "by")
}

// isEnumKey diz se a chave guarda um código de domínio (taskStatus, type...).
func isEnumKey(k string) bool {
	return enumKeys[k] || strings.HasSuffix(k, "status") || strings.HasSuffix(k, "type")
}

// Anonymize decodifica body, anonimiza e devolve JSON indentado.
func (a *anonymizer) Anonymize(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("resposta não é JSON: %v", err)
	}
	out, err := json.MarshalIndent(a.walk(v, ""), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func (a *anonymizer) walk(v any, key string) any {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(val))
		for _, k := range keys {
			newKey := k
			if reDigits.MatchString(k) && k != "0" {
				newKey = a.remapID(k)
			}
			out[newKey] = a.walk(val[k], k)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			// O elemento herda a chave do array: tagIds: [1, 2] continua
			// sendo lista de IDs.
			out[i] = a.walk(item, key)
		}
		return out
	case json.Number:
		if isIDKey(key) && reDigits.MatchString(val.String()) && val.String() != "0" {
			return json.Number(a.remapID(val.String()))
		}
		return val
	case string:
		return a.anonString(val, key)
	default:
		return v // bool e null
	}
}

func (a *anonymizer) remapID(original string) string {
	if fake, ok := a.ids[original]; ok {
		return fake
	}
	a.nextID++
	fake := fmt.Sprintf("%d", a.nextID)
	a.ids[original] = fake
	return fake
}

func (a *anonymizer) anonString(s, key string) string {
	if s == "" {
		return s
	}
	if isIDKey(key) && reDigits.MatchString(s) {
		if s == "0" {
			return s
		}
		return a.remapID(s)
	}
	if reNumeric.MatchString(s) || looksLikeDate(s) {
		return s
	}
	k := strings.ToLower(key)
	if (isEnumKey(k) || reHexCor.MatchString(s)) && len(s) <= 40 &&
		!strings.ContainsAny(s, "@/ ") {
		if a.kept[key] == nil {
			a.kept[key] = map[string]bool{}
		}
		a.kept[key][s] = true
		return s
	}
	return a.fake(s, k)
}

// fake devolve o mesmo valor fictício para o mesmo texto original e categoria.
func (a *anonymizer) fake(s, k string) string {
	cat := "texto"
	switch {
	case strings.Contains(k, "email"):
		cat = "email"
	case strings.Contains(k, "url"), strings.Contains(k, "avatar"),
		strings.Contains(k, "host"), strings.Contains(k, "link"),
		strings.Contains(k, "photo"), strings.Contains(k, "image"),
		strings.Contains(k, "logo"), strings.Contains(k, "website"),
		strings.Contains(s, "://"):
		cat = "url"
	case strings.Contains(k, "description"), strings.Contains(k, "content"),
		strings.Contains(k, "title"), strings.Contains(k, "message"):
		cat = "descricao"
	case strings.Contains(k, "name"):
		cat = "nome"
	}

	mapKey := cat + "\x00" + s
	if fake, ok := a.strs[mapKey]; ok {
		return fake
	}
	a.counter[cat]++
	n := a.counter[cat]
	var fake string
	switch cat {
	case "email":
		fake = fmt.Sprintf("pessoa%d@exemplo.invalid", n)
	case "url":
		fake = fmt.Sprintf("https://exemplo.invalid/recurso/%d", n)
	case "descricao":
		fake = fmt.Sprintf("Descricao ficticia %d", n)
	case "nome":
		fake = fmt.Sprintf("Nome ficticio %d", n)
	default:
		fake = fmt.Sprintf("Texto ficticio %d", n)
	}
	a.strs[mapKey] = fake
	return fake
}

// KeptSummary lista, por chave, os textos que foram mantidos como estavam.
func (a *anonymizer) KeptSummary() []string {
	keys := make([]string, 0, len(a.kept))
	for k := range a.kept {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := make([]string, 0, len(a.kept[k]))
		for v := range a.kept[k] {
			vals = append(vals, v)
		}
		sort.Strings(vals)
		lines = append(lines, fmt.Sprintf("%s: %s", k, strings.Join(vals, ", ")))
	}
	return lines
}

// findLeaks procura, sem diferenciar maiúsculas, cada termo sensível no texto.
func findLeaks(data []byte, needles []string) []string {
	lower := strings.ToLower(string(data))
	var found []string
	for _, n := range needles {
		n = strings.ToLower(strings.TrimSpace(n))
		if len(n) < 3 {
			continue
		}
		if strings.Contains(lower, n) {
			found = append(found, n)
		}
	}
	return found
}
