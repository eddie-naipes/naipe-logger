package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeHostRejectsInsecureAndInvalid(t *testing.T) {
	cases := []struct {
		name string
		host string
	}{
		{"http explícito", "http://teamwork.empresa.com.br"},
		{"http maiúsculo", "HTTP://teamwork.empresa.com.br"},
		{"esquema não http", "ftp://teamwork.empresa.com.br"},
		{"vazio", ""},
		{"só espaços", "   "},
		{"credenciais embutidas", "https://user:senha@teamwork.empresa.com.br"},
		{"com caminho", "https://teamwork.empresa.com.br/projects"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeHost(tc.host)
			if err == nil {
				t.Fatalf("NormalizeHost(%q) = %q, esperava erro", tc.host, got)
			}
		})
	}
}

func TestNormalizeHostCanonicalizes(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"teamwork.empresa.com.br", "https://teamwork.empresa.com.br"},
		{"https://teamwork.empresa.com.br", "https://teamwork.empresa.com.br"},
		{"https://teamwork.empresa.com.br/", "https://teamwork.empresa.com.br"},
		{"  teamwork.empresa.com.br  ", "https://teamwork.empresa.com.br"},
		{"empresa.teamwork.com:8443", "https://empresa.teamwork.com:8443"},
	}

	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			got, err := NormalizeHost(tc.host)
			if err != nil {
				t.Fatalf("NormalizeHost(%q) devolveu erro: %v", tc.host, err)
			}
			if got != tc.want {
				t.Errorf("NormalizeHost(%q) = %q, esperava %q", tc.host, got, tc.want)
			}
		})
	}
}

// createRequest é a única porta por onde o token sai da aplicação. Se ela
// aceitasse http, o Basic auth exporia o segredo em claro na rede.
func TestCreateRequestRecusaNaoHTTPS(t *testing.T) {
	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "https://x.com"}}

	if _, err := api.createRequest("GET", "http://x.com/me.json", nil); err == nil {
		t.Fatal("createRequest aceitou http://, deveria recusar")
	}

	if _, err := api.createRequest("GET", "https://x.com/me.json", nil); err != nil {
		t.Fatalf("createRequest recusou https://: %v", err)
	}
}

// Se o host for inválido, buildURL não pode devolver uma URL utilizável, senão
// a requisição sairia sem a proteção de https.
func TestBuildURLVazioParaHostInvalido(t *testing.T) {
	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "http://x.com"}}

	if api.IsConfigured() {
		t.Error("IsConfigured() = true para host http://, esperava false")
	}
	if url := api.buildURL("/me.json"); strings.HasPrefix(url, "http://") {
		t.Errorf("buildURL devolveu URL insegura: %q", url)
	}
}

// O token não pode atravessar nenhuma serialização: nem para config.json, nem
// para o frontend via Wails.
func TestConfigNaoSerializaToken(t *testing.T) {
	data, err := json.Marshal(Config{
		AuthToken: "tok-super-secreto",
		UserID:    42,
		ApiHost:   "https://teamwork.empresa.com.br",
	})
	if err != nil {
		t.Fatalf("Marshal falhou: %v", err)
	}

	if strings.Contains(string(data), "tok-super-secreto") {
		t.Errorf("token vazou na serialização de Config: %s", data)
	}
	if strings.Contains(string(data), "authToken") {
		t.Errorf("campo authToken presente na serialização de Config: %s", data)
	}
}

func TestLoginResponseNaoSerializaToken(t *testing.T) {
	data, err := json.Marshal(LoginResponse{Success: true, UserID: 42, InstanceID: "https://x.com"})
	if err != nil {
		t.Fatalf("Marshal falhou: %v", err)
	}
	if strings.Contains(strings.ToLower(string(data)), "token") {
		t.Errorf("LoginResponse expõe campo de token: %s", data)
	}
}

func TestValidateTokenRecusaEmailSenha(t *testing.T) {
	_, err := ValidateToken("usuario@empresa.com:senha123", "teamwork.empresa.com.br")
	if err == nil {
		t.Fatal("ValidateToken aceitou email:senha, deveria recusar")
	}
}
