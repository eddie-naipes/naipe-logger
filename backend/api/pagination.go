package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	// maxListPages limita a paginação de tarefas e projetos. Com pageSize=250
	// cobre 5 mil itens, muito acima do que um usuário tem atribuído.
	maxListPages = 20
	// listPageSize é o tamanho de página pedido nas listagens v3; o padrão da
	// API (50) cortava silenciosamente quem tinha mais tarefas que isso.
	listPageSize = 250
	// timeEntryPageSize é o tamanho de página das listagens de lançamentos.
	timeEntryPageSize = 500
)

// pageMeta é o bloco de paginação que os endpoints v2/v3 devolvem. HasMore é
// ponteiro para distinguir "a API disse que acabou" de "a API não informou".
type pageMeta struct {
	Meta struct {
		Page struct {
			HasMore    *bool `json:"hasMore"`
			TotalItems int   `json:"totalItems"`
		} `json:"page"`
	} `json:"meta"`
}

// pageInfo resume uma página já decodificada para fetchPages decidir se segue.
type pageInfo struct {
	items   int
	hasMore *bool
}

// fetchPages percorre um endpoint paginado chamando handle para cada página.
//
// Para quando a página vem vazia, quando a API diz que não há mais (meta
// hasMore ou cabeçalho X-Pages) ou, se ela não diz nada, quando a página vem
// incompleta. maxPages é a rede de segurança contra uma API que devolva
// hasMore=true para sempre. Erros em qualquer página são propagados: devolver
// só parte dos dados como se fosse o todo esconde lançamentos do usuário.
func (t *TeamworkAPI) fetchPages(baseURL string, pageSize, maxPages int, what string,
	handle func(body []byte) (pageInfo, error)) error {

	for page := 1; page <= maxPages; page++ {
		pageURL := withPagination(baseURL, page, pageSize)

		req, err := t.createRequest("GET", pageURL, nil)
		if err != nil {
			return err
		}

		resp, body, err := t.doRequest(req)
		if err != nil {
			return fmt.Errorf("erro ao obter %s (página %d): %v", what, page, err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("erro ao obter %s (página %d): %d %s - %s",
				what, page, resp.StatusCode, resp.Status, truncateForError(body, 200))
		}

		info, err := handle(body)
		if err != nil {
			return fmt.Errorf("erro ao decodificar %s (página %d): %v", what, page, err)
		}

		if info.items == 0 {
			return nil
		}

		if !pageHasMore(info, resp, page, pageSize) {
			return nil
		}

		if page == maxPages {
			t.logWarn("Limite de %d páginas atingido ao obter %s; podem existir mais itens", maxPages, what)
		}
	}

	return nil
}

func pageHasMore(info pageInfo, resp *http.Response, page, pageSize int) bool {
	if info.hasMore != nil {
		return *info.hasMore
	}
	// Endpoints v1/v2 informam o total de páginas no cabeçalho.
	if pages, err := strconv.Atoi(resp.Header.Get("X-Pages")); err == nil && pages > 0 {
		return page < pages
	}
	return info.items >= pageSize
}

// withPagination acrescenta page e pageSize à URL. A URL base não deve trazer
// esses parâmetros.
func withPagination(baseURL string, page, pageSize int) string {
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%spage=%d&pageSize=%d", baseURL, sep, page, pageSize)
}

func truncateForError(body []byte, limit int) string {
	return string(body[:min(len(body), limit)])
}
