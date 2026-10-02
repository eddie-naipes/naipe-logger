package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestAPI sobe um servidor TLS de teste e devolve um cliente configurado
// para ele. createRequest só aceita https, então o servidor precisa ser TLS e
// o cliente precisa confiar no certificado dele.
func newTestAPI(t *testing.T, handler http.HandlerFunc) (*TeamworkAPI, *httptest.Server) {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	api := NewTeamworkAPI(Config{
		AuthToken: "token-de-teste",
		UserID:    42,
		ApiHost:   server.URL,
	})
	api.httpClient = server.Client()
	return api, server
}
