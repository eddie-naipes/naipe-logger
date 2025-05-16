package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func (t *TeamworkAPI) GetProjects() ([]Project, error) {
	cacheKey := "projects"
	if cachedData, found := t.cache.Get(cacheKey); found {
		return cachedData.([]Project), nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	path := "/projects/api/v3/projects.json?includeProjectUserInfo=true&include=tags,projectTaskStats,projectCategories,companies&projectStatuses=active"
	url := t.buildURL(path)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter projetos: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response ProjectsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v\nBody: %s", err, string(body))
	}

	t.cache.Set(cacheKey, response.Projects, 30*time.Minute)
	return response.Projects, nil
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

	return response.TotalItems, nil
}

func (t *TeamworkAPI) GetHoursLogToProject(projectID int, startDate, endDate string) (float64, error) {
	projectIDStr := strconv.Itoa(projectID)
	path := fmt.Sprintf("/projects/api/v3/time.json?projectIds=%s&fromDate=%s&toDate=%s",
		projectIDStr, startDate, endDate)
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
		return 0, fmt.Errorf("erro ao obter horas do projeto: %d", resp.StatusCode)
	}

	var responseData struct {
		TimeEntries []struct {
			Minutes float64 `json:"minutes"`
		} `json:"timeEntries"`
	}

	if err := json.Unmarshal(body, &responseData); err != nil {
		return 0, err
	}

	totalMinutos := 0.0
	for _, entry := range responseData.TimeEntries {
		totalMinutos += entry.Minutes
	}

	return totalMinutos / 60.0, nil
}

func getProjectColor(id int) string {
	colors := []string{
		"bg-blue-500",
		"bg-green-500",
		"bg-purple-500",
		"bg-amber-500",
		"bg-red-500",
		"bg-indigo-500",
		"bg-pink-500",
		"bg-emerald-500",
	}
	return colors[id%len(colors)]
}
