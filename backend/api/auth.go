package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (t *TeamworkAPI) GetCurrentUserId() (int, error) {
	if t.Config.AuthToken == "" || t.Config.ApiHost == "" {
		return 0, fmt.Errorf("API não configurada (falta token ou host)")
	}

	path := "/projects/api/v3/me.json"
	url := t.buildURL(path)

	t.logDebug("Consultando API do Teamwork em: %s", url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return 0, err
	}

	t.logDebug("Resposta da API (primeiros 500 caracteres): %s", string(body[:minValue(len(body), 500)]))

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("erro ao obter informações do usuário: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	userID, found := extractUserIDFromResponse(body)
	if !found {
		return 0, fmt.Errorf("não foi possível encontrar o ID do usuário na resposta")
	}

	return userID, nil
}

func extractUserIDFromResponse(body []byte) (int, bool) {
	var responsePerson struct {
		Person struct {
			ID int `json:"id"`
		} `json:"person"`
	}

	err := json.Unmarshal(body, &responsePerson)
	if err == nil && responsePerson.Person.ID > 0 {
		return responsePerson.Person.ID, true
	}

	var responseUser struct {
		User struct {
			ID int `json:"id"`
		} `json:"user"`
	}

	err = json.Unmarshal(body, &responseUser)
	if err == nil && responseUser.User.ID > 0 {
		return responseUser.User.ID, true
	}

	var responseAlt map[string]interface{}
	err = json.Unmarshal(body, &responseAlt)
	if err != nil {
		return 0, false
	}

	return findUserIdInMap(responseAlt)
}

func findUserIdInMap(data map[string]interface{}) (int, bool) {
	if id, ok := data["id"]; ok {
		if idInt, ok := id.(float64); ok {
			return int(idInt), true
		}
	}

	if person, ok := data["person"].(map[string]interface{}); ok {
		if id, ok := person["id"]; ok {
			if idInt, ok := id.(float64); ok {
				return int(idInt), true
			}
		}
	}

	if user, ok := data["user"].(map[string]interface{}); ok {
		if id, ok := user["id"]; ok {
			if idInt, ok := id.(float64); ok {
				return int(idInt), true
			}
		}
	}

	if me, ok := data["me"].(map[string]interface{}); ok {
		if id, ok := me["id"]; ok {
			if idInt, ok := id.(float64); ok {
				return int(idInt), true
			}
		}
	}

	return 0, false
}

func (t *TeamworkAPI) GetTokenWithCredentials(email, password, host string) (*LoginResponse, error) {
	if email == "" || password == "" || host == "" {
		return nil, fmt.Errorf("email, senha e host são obrigatórios")
	}

	baseURL := host
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	url := fmt.Sprintf("%s/projects/api/v3/me.json", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	if resp.StatusCode != 200 {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("Erro na autenticação: %d %s - %s",
				resp.StatusCode, resp.Status, string(body)),
		}, nil
	}

	userID, found := extractUserIDFromResponse(body)
	if !found {
		userID = 0
	}

	t.logDebug("Resposta da API ME: %s", string(body))

	result := &LoginResponse{
		Success:    true,
		Token:      email + ":" + password,
		UserID:     userID,
		InstanceID: host,
		Message:    "Autenticação bem-sucedida",
	}

	return result, nil
}

func GetTokenWithCredentials(email, password, host string) (*LoginResponse, error) {
	tempAPI := &TeamworkAPI{}
	return tempAPI.GetTokenWithCredentials(email, password, host)
}

func minValue(a, b int) int {
	if a < b {
		return a
	}
	return b
}
