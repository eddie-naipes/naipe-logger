package api

import (
	"encoding/json"
	"fmt"
	"time"
)

func (t *TeamworkAPI) GetProjects() ([]Project, error) {
	cacheKey := "projects"
	if cached, found := getCached[[]Project](t.cache, cacheKey); found {
		return cached, nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	path := "/projects/api/v3/projects.json?includeProjectUserInfo=true&include=tags,projectTaskStats,projectCategories,companies&projectStatuses=active"

	// Sem paginar, só a primeira página (50 projetos, o padrão da v3) chegava
	// ao seletor de projetos.
	projects := make([]Project, 0)
	err := t.fetchPages(t.buildURL(path), listPageSize, maxListPages, "projetos",
		func(body []byte) (pageInfo, error) {
			var page struct {
				Projects []Project `json:"projects"`
				pageMeta
			}
			if err := json.Unmarshal(body, &page); err != nil {
				return pageInfo{}, err
			}
			projects = append(projects, page.Projects...)
			return pageInfo{items: len(page.Projects), hasMore: page.Meta.Page.HasMore}, nil
		})
	if err != nil {
		return nil, err
	}

	t.cache.Set(cacheKey, projects, 30*time.Minute)
	return projects, nil
}

func (t *TeamworkAPI) GetProjectCount() (int, error) {
	if !t.IsConfigured() {
		return 0, fmt.Errorf("API não configurada")
	}

	path := "/projects/api/v3/projects.json?projectStatuses=active&page=1&pageSize=1"
	url := t.buildURL(path)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("erro ao obter projetos: %d %s", resp.StatusCode, resp.Status)
	}

	var response ProjectsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	// A v3 informa o total em meta.page.totalItems, como em GetTaskCount. O
	// campo no topo só existia em formatos antigos e vinha zerado.
	if response.Meta.Page.TotalItems > 0 {
		return response.Meta.Page.TotalItems, nil
	}
	return response.TotalItems, nil
}
