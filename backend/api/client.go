package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"logTime-go/backend/logging"
)

var (
	httpClient     *http.Client
	once           sync.Once
	downloadClient *http.Client
	downloadOnce   sync.Once
)

func (t *TeamworkAPI) IsConfigured() bool {
	return t.Config.AuthToken != "" && t.BaseURL() != ""
}

// BaseURL devolve o host da API já normalizado para https. Hosts inválidos ou
// http são reduzidos a "" para que createRequest recuse a requisição em vez de
// vazar o token em claro.
//
// A normalização é feita uma vez em NewTeamworkAPI. O host bruto que a gerou
// fica guardado: se Config.ApiHost for trocado depois (ou se o cliente foi
// montado sem o construtor, como em ValidateToken e nos testes), o valor é
// recalculado em vez de servir um host desatualizado.
func (t *TeamworkAPI) BaseURL() string {
	if t.hostNormalized && t.normalizedFrom == t.Config.ApiHost {
		return t.baseURL
	}
	return normalizeHostOrEmpty(t.Config.ApiHost)
}

func normalizeHostOrEmpty(host string) string {
	normalized, err := NormalizeHost(host)
	if err != nil {
		return ""
	}
	return normalized
}

func (t *TeamworkAPI) buildURL(path string) string {
	return t.BaseURL() + path
}

func (t *TeamworkAPI) createRequest(method, rawURL string, body io.Reader) (*http.Request, error) {
	// Última barreira antes de enviar o token: o Basic auth expõe o segredo em
	// base64 no cabeçalho, então nenhuma requisição sai fora de https.
	if !strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return nil, fmt.Errorf("requisição bloqueada: apenas https é permitido, recebido %q", rawURL)
	}

	// O contexto da aplicação viaja com a requisição: fechar o aplicativo
	// cancela o que estiver em voo em vez de esperar o timeout do cliente.
	req, err := http.NewRequestWithContext(t.requestContext(), method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Tokens de API do Teamwork são usados como usuário do Basic auth, com
	// qualquer valor como senha.
	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func getHTTPClient() *http.Client {
	once.Do(func() {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
				DisableCompression:  false,
			},
		}
	})
	return httpClient
}

// client devolve o cliente HTTP das chamadas de API. Os testes injetam o
// cliente de um httptest.NewTLSServer, já que createRequest só aceita https.
func (t *TeamworkAPI) client() *http.Client {
	if t.httpClient != nil {
		return t.httpClient
	}
	return getHTTPClient()
}

// getDownloadClient serve downloads de relatório, que podem levar bem mais que
// o timeout curto usado nas chamadas de API.
func getDownloadClient() *http.Client {
	downloadOnce.Do(func() {
		downloadClient = &http.Client{
			Timeout:   2 * time.Minute,
			Transport: getHTTPClient().Transport,
		}
	})
	return downloadClient
}

// GetJSON faz um GET autenticado num caminho da API e devolve corpo e status.
// Existe para que quem está fora do pacote não precise montar o cabeçalho de
// autenticação — e portanto não precise enxergar o token.
func (t *TeamworkAPI) GetJSON(path string) ([]byte, int, error) {
	if !t.IsConfigured() {
		return nil, 0, fmt.Errorf("API não configurada")
	}

	req, err := t.createRequest("GET", t.buildURL(path), nil)
	if err != nil {
		return nil, 0, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, 0, err
	}

	return body, resp.StatusCode, nil
}

// doRequest executa a requisição, repetindo em rate limit (429) e, para métodos
// idempotentes, em falha de rede ou erro 5xx. Sem isso um lote grande de
// lançamentos era abandonado inteiro no primeiro 429 do Teamwork.
func (t *TeamworkAPI) doRequest(req *http.Request) (*http.Response, []byte, error) {
	client := t.client()
	idempotent := isIdempotent(req.Method)

	var retryAfter time.Duration
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			espera := backoffDuration(attempt, retryAfter)
			slog.Debug("Repetindo requisição", "tentativa", attempt+1, "max", maxRetries+1,
				"metodo", req.Method, "caminho", req.URL.Path, "espera", espera)
			if err := sleepContext(req.Context(), espera); err != nil {
				if lastErr != nil {
					return nil, nil, fmt.Errorf("%v; espera interrompida: %v", lastErr, err)
				}
				return nil, nil, fmt.Errorf("espera interrompida: %v", err)
			}
			retryAfter = 0
		}

		attemptReq := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, nil, fmt.Errorf("erro ao preparar corpo da requisição: %v", err)
			}
			attemptReq.Body = body
		}

		resp, err := client.Do(attemptReq)
		if err != nil {
			lastErr = fmt.Errorf("erro na requisição: %v", err)
			// Numa falha de rede em POST não dá para saber se o servidor
			// processou; repetir poderia duplicar o lançamento.
			if !idempotent || attempt == maxRetries {
				return nil, nil, lastErr
			}
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if readErr != nil {
			lastErr = fmt.Errorf("erro ao ler resposta: %v", readErr)
			if !idempotent || attempt == maxRetries {
				return resp, nil, lastErr
			}
			continue
		}

		if attempt < maxRetries && shouldRetryStatus(resp.StatusCode, idempotent) {
			retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			lastErr = fmt.Errorf("resposta %d %s", resp.StatusCode, resp.Status)
			continue
		}

		return resp, body, nil
	}

	return nil, nil, lastErr
}

// sanitizeForLog mascara um valor antes de ele ir para o log. O handler de
// backend/logging já aplica a mesma regra a todo atributo; chamar aqui deixa
// explícito, nos pontos que logam corpos de resposta, que eles podem conter
// dados sensíveis.
func sanitizeForLog(data interface{}) interface{} {
	if str, ok := data.(string); ok {
		return logging.Sanitize(str)
	}
	return data
}

// registerSecretForLogs avisa ao logger que o token (e sua forma em Basic
// auth, que é o que vai no cabeçalho) nunca pode aparecer em log. Chamado em
// todo ponto onde um token entra no cliente.
func registerSecretForLogs(token string) {
	if token == "" {
		return
	}
	logging.AddSecret(token)
	logging.AddSecret(base64.StdEncoding.EncodeToString([]byte(token + ":X")))
}
