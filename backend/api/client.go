package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	httpClient *http.Client
	once       sync.Once
)

func (t *TeamworkAPI) IsConfigured() bool {
	return t.Config.AuthToken != "" && t.Config.ApiHost != ""
}

func (t *TeamworkAPI) buildURL(path string) string {
	baseURL := t.Config.ApiHost
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	return fmt.Sprintf("%s%s", baseURL, path)
}

func (t *TeamworkAPI) createRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

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

func (t *TeamworkAPI) doRequest(req *http.Request) (*http.Response, []byte, error) {
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("erro na requisição: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	return resp, body, nil
}

func (t *TeamworkAPI) TestConnection(config Config) (bool, string) {
	tempAPI := &TeamworkAPI{
		Config: config,
	}

	if !tempAPI.IsConfigured() {
		return false, "API não configurada"
	}

	url := tempAPI.buildURL("/projects/api/v3/me.json")
	req, err := tempAPI.createRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("Erro ao criar requisição: %v", err)
	}

	resp, _, err := tempAPI.doRequest(req)
	if err != nil {
		return false, fmt.Sprintf("Erro na requisição: %v", err)
	}

	if resp.StatusCode != 200 {
		return false, fmt.Sprintf("Autenticação falhou: %d %s", resp.StatusCode, resp.Status)
	}

	return true, "Conexão estabelecida com sucesso!"
}

func (t *TeamworkAPI) logDebug(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
