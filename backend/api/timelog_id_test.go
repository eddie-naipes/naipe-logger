package api

import "testing"

// Sem o ID devolvido pelo POST o lançamento existe no Teamwork mas não pode ser
// desfeito pela ferramenta. O formato da resposta varia por versão do endpoint,
// então cada forma conhecida precisa continuar sendo reconhecida.
func TestExtractTimelogID(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"id no topo", `{"id": 12345, "status": "OK"}`, 12345},
		{"timelogId no topo", `{"timelogId": 999}`, 999},
		{"aninhado em timelog", `{"timelog": {"id": 777, "minutes": 60}}`, 777},
		{"aninhado em timeEntry", `{"timeEntry": {"id": 555}}`, 555},
		{"aninhado em timeLog", `{"timeLog": {"id": 444}}`, 444},
		{"id zero e ignorado", `{"id": 0, "timelog": {"id": 321}}`, 321},
		{"sem id", `{"status": "OK"}`, 0},
		{"corpo vazio", ``, 0},
		{"json invalido", `nao e json`, 0},
		{"array", `[1,2,3]`, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := extractTimelogID([]byte(tc.body))

			if tc.want == 0 {
				if found {
					t.Errorf("extractTimelogID(%s) = %d, esperava não encontrar", tc.body, got)
				}
				return
			}

			if !found {
				t.Fatalf("extractTimelogID(%s) não encontrou ID, esperava %d", tc.body, tc.want)
			}
			if got != tc.want {
				t.Errorf("extractTimelogID(%s) = %d, esperava %d", tc.body, got, tc.want)
			}
		})
	}
}
