package api

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeHost valida e canonicaliza o host da API do Teamwork, devolvendo-o
// sempre no formato "https://host".
//
// Apenas https é aceito: a autenticação é Basic, ou seja, o token viaja em
// base64 no cabeçalho de toda requisição. Sobre http ele trafegaria em claro.
func NormalizeHost(host string) (string, error) {
	h := strings.TrimSpace(host)
	if h == "" {
		return "", fmt.Errorf("host da API não informado")
	}

	if strings.HasPrefix(strings.ToLower(h), "http://") {
		return "", fmt.Errorf("host inseguro %q: http:// não é aceito porque exporia o token de API na rede, use https://", host)
	}

	// Aceita tanto "https://empresa.teamwork.com" quanto "empresa.teamwork.com".
	if i := strings.Index(h, "://"); i >= 0 {
		if !strings.EqualFold(h[:i], "https") {
			return "", fmt.Errorf("host da API inválido %q: apenas https é aceito", host)
		}
		h = h[i+3:]
	}
	h = strings.Trim(h, "/")

	parsed, err := url.Parse("https://" + h)
	if err != nil {
		return "", fmt.Errorf("host da API inválido %q: %v", host, err)
	}
	if parsed.Host == "" || parsed.User != nil || strings.ContainsAny(h, " \t\r\n") {
		return "", fmt.Errorf("host da API inválido: %q", host)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("host da API inválido %q: informe apenas o domínio, sem caminho", host)
	}

	return "https://" + parsed.Host, nil
}
