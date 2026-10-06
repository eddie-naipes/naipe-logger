package api

import "net/http"

// SetHTTPClient troca o cliente HTTP usado nas chamadas de API. Existe para
// testes de outros pacotes, que sobem um httptest.NewTLSServer e precisam
// confiar no certificado dele (createRequest só aceita https).
func (t *TeamworkAPI) SetHTTPClient(c *http.Client) {
	t.httpClient = c
}
