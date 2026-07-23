package api

import (
	"encoding/json"
	"fmt"
	"strings"
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

// ValidateToken confere um token de API do Teamwork contra o host informado e
// devolve o ID do usuário dono do token. O token não é persistido aqui — quem
// chama decide o que fazer com ele (ver security.StoreToken).
func ValidateToken(token, host string) (*LoginResponse, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token de API é obrigatório")
	}
	if strings.Contains(token, ":") {
		return nil, fmt.Errorf("isto não parece um token de API do Teamwork; não use email:senha, gere um token no seu perfil do Teamwork (Edit My Details > API & Mobile)")
	}

	baseURL, err := NormalizeHost(host)
	if err != nil {
		return nil, err
	}

	tempAPI := &TeamworkAPI{Config: Config{AuthToken: token, ApiHost: baseURL}}

	req, err := tempAPI.createRequest("GET", baseURL+"/projects/api/v3/me.json", nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := tempAPI.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return &LoginResponse{
			Success: false,
			Message: "Token de API rejeitado pelo Teamwork. Confira o token e o domínio da empresa.",
		}, nil
	}
	if resp.StatusCode != 200 {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("Erro na autenticação: %d %s", resp.StatusCode, resp.Status),
		}, nil
	}

	userID, found := extractUserIDFromResponse(body)
	if !found || userID <= 0 {
		return &LoginResponse{
			Success: false,
			Message: "Autenticado, mas não foi possível identificar o usuário do token.",
		}, nil
	}

	return &LoginResponse{
		Success:    true,
		UserID:     userID,
		InstanceID: baseURL,
		Message:    "Token validado com sucesso",
	}, nil
}

func minValue(a, b int) int {
	if a < b {
		return a
	}
	return b
}
