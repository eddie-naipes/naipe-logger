package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnonymizeTrocaDadosPessoaisEPreservaEstrutura(t *testing.T) {
	body := `{
		"person": {"id": 123456, "firstName": "Fulano", "lastName": "Silva", "email": "fulano@empresa.com",
			"avatarUrl": "https://empresa.teamwork.com/a.png", "company": {"id": "77", "name": "Empresa X"}},
		"tasks": [{"id": 5, "name": "Corrigir login", "tasklistId": 9, "status": "new",
			"assignees": [{"id": 123456, "type": "users"}], "tagIds": [3, 0], "dueDate": "2026-09-30T00:00:00Z"}],
		"included": {"tasklists": {"9": {"id": 9, "name": "Sprint 1"}}},
		"billable": [["1790812800000", 0.25, 15]],
		"meta": {"page": {"count": 12, "hasMore": true}}
	}`

	a := newAnonymizer()
	out, err := a.Anonymize([]byte(body))
	if err != nil {
		t.Fatalf("Anonymize: %v", err)
	}
	s := string(out)

	for _, sensivel := range []string{"Fulano", "Silva", "empresa", "Corrigir", "Sprint", "123456", "Empresa X"} {
		if strings.Contains(s, sensivel) {
			t.Errorf("saída ainda contém %q:\n%s", sensivel, s)
		}
	}

	var r struct {
		Person struct {
			ID      int `json:"id"`
			Company struct {
				ID string `json:"id"`
			} `json:"company"`
		} `json:"person"`
		Tasks []struct {
			TasklistID int    `json:"tasklistId"`
			Status     string `json:"status"`
			DueDate    string `json:"dueDate"`
			TagIDs     []int  `json:"tagIds"`
			Assignees  []struct {
				ID   int    `json:"id"`
				Type string `json:"type"`
			} `json:"assignees"`
		} `json:"tasks"`
		Included struct {
			Tasklists map[string]struct {
				ID int `json:"id"`
			} `json:"tasklists"`
		} `json:"included"`
		Billable [][3]any `json:"billable"`
		Meta     struct {
			Page struct {
				Count int `json:"count"`
			} `json:"page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("saída não decodifica nos tipos originais: %v\n%s", err, s)
	}

	task := r.Tasks[0]
	if task.Assignees[0].ID != r.Person.ID {
		t.Error("o mesmo ID original deveria virar o mesmo ID fictício")
	}
	if task.Assignees[0].Type != "users" || task.Status != "new" {
		t.Error("enumerações deveriam ser mantidas")
	}
	if task.DueDate != "2026-09-30T00:00:00Z" {
		t.Errorf("data alterada: %q", task.DueDate)
	}
	if task.TagIDs[1] != 0 {
		t.Error("ID zero deveria ser mantido")
	}
	if r.Person.Company.ID == "77" || r.Person.Company.ID == "" {
		t.Errorf("ID em texto deveria ser remapeado como texto, veio %q", r.Person.Company.ID)
	}
	if _, ok := r.Included.Tasklists[itoa(task.TasklistID)]; !ok {
		t.Error("chave numérica de included deveria acompanhar o remapeamento")
	}
	if r.Meta.Page.Count != 12 {
		t.Error("meta de paginação deveria ser mantida")
	}
	if r.Billable[0][0] != "1790812800000" || r.Billable[0][1] != 0.25 || r.Billable[0][2] != float64(15) {
		t.Errorf("dia do loggedtime alterado: %v", r.Billable[0])
	}

	// Determinismo: a mesma entrada gera a mesma saída.
	out2, _ := newAnonymizer().Anonymize([]byte(body))
	if string(out2) != s {
		t.Error("anonimização não é determinística")
	}
}

func TestFindLeaksIgnoraCaixaETermosCurtos(t *testing.T) {
	got := findLeaks([]byte(`{"a":"FULANO de tal"}`), []string{"fulano", "de", "beltrano"})
	if len(got) != 1 || got[0] != "fulano" {
		t.Errorf("findLeaks = %v", got)
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
