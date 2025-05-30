package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type TimeTotal struct {
	FinancialTotals struct {
		TotalCost         float64 `json:"totalCost"`
		TotalCostBillable float64 `json:"totalCostBillable"`
		TotalCostBilled   float64 `json:"totalCostBilled"`
	} `json:"financialTotals"`
	SubTasks struct {
		EstimatedMinutes int `json:"estimatedMinutes"`
		Minutes          int `json:"minutes"`
		MinutesBillable  int `json:"minutesBillable"`
	} `json:"subTasks"`
	TimeTotals struct {
		EstimatedMinutes               int `json:"estimatedMinutes"`
		EstimatedMinutesActive         int `json:"estimatedMinutesActive"`
		EstimatedMinutesCompleted      int `json:"estimatedMinutesCompleted"`
		EstimatedMinutesFiltered       int `json:"estimatedMinutesFiltered"`
		EstimatedMinutesWithLoggedTime int `json:"estimatedMinutesWithLoggedTime"`
		Minutes                        int `json:"minutes"`
		MinutesBillable                int `json:"minutesBillable"`
		MinutesBilled                  int `json:"minutesBilled"`
		MinutesNonBillable             int `json:"minutesNonBillable"`
		MinutesNonBilled               int `json:"minutesNonBilled"`
	} `json:"time-totals"`
}

type TimeEntriesResponse struct {
	TimeEntries []TimeEntryReport `json:"timeEntries"`
	Meta        struct {
		Page struct {
			Count      int  `json:"count"`
			HasMore    bool `json:"hasMore"`
			TotalItems int  `json:"totalItems"`
		} `json:"page"`
	} `json:"meta"`
}

type LoggedTimeResponse struct {
	STATUS string `json:"STATUS"`
	User   struct {
		Billable    [][3]string `json:"billable"`
		Firstname   string      `json:"firstname"`
		Lastname    string      `json:"lastname"`
		Nonbillable [][3]string `json:"nonbillable"`
		ID          string      `json:"id"`
		Endepoch    string      `json:"endepoch"`
		Startepoch  string      `json:"startepoch"`
	} `json:"user"`
}

func (t *TeamworkAPI) GetDefaultReportPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório do usuário: %v", err)
	}

	now := time.Now()
	monthYear := now.Format("2006-01")

	reportsDir := filepath.Join(homeDir, "TeamworkReports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return "", fmt.Errorf("erro ao criar diretório de relatórios: %v", err)
	}

	fileName := fmt.Sprintf("TeamworkReport_%s.pdf", monthYear)
	return filepath.Join(reportsDir, fileName), nil
}

func (t *TeamworkAPI) GetTimeEntriesForPeriod(startDate, endDate string) ([]TimeEntryReport, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("data inicial inválida: %v", err)
	}

	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("data final inválida: %v", err)
	}

	path := fmt.Sprintf("/projects/api/v3/time.json?startDate=%s&endDate=%s&userId=%d&pageSize=500",
		startDate, endDate, t.Config.UserID)
	url := t.buildURL(path)

	t.logDebug("Obtendo entradas de tempo de %s a %s...", startDate, endDate)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter entradas de tempo: %d %s - %s",
			resp.StatusCode, resp.Status, string(body[:minValue(len(body), 100)]))
	}

	var response TimeEntriesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	allEntries := response.TimeEntries
	page := 2
	for response.Meta.Page.HasMore {
		pageUrl := fmt.Sprintf("%s&page=%d", url, page)
		req, err := t.createRequest("GET", pageUrl, nil)
		if err != nil {
			break
		}

		resp, body, err := t.doRequest(req)
		if err != nil || resp.StatusCode != 200 {
			break
		}

		if err := json.Unmarshal(body, &response); err != nil {
			break
		}

		allEntries = append(allEntries, response.TimeEntries...)
		page++
	}

	return allEntries, nil
}

func (t *TeamworkAPI) GetTimeTotalsForPeriod(startDate, endDate string) (*TimeTotal, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("data inicial inválida: %v", err)
	}

	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("data final inválida: %v", err)
	}

	path := fmt.Sprintf("/projects/api/v3/time/total.json?startDate=%s&endDate=%s&userId=%d",
		startDate, endDate, t.Config.UserID)
	url := t.buildURL(path)

	t.logDebug("Obtendo totais de tempo de %s a %s...", startDate, endDate)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter totais de tempo: %d %s - %s",
			resp.StatusCode, resp.Status, string(body[:minValue(len(body), 100)]))
	}

	var timeTotal TimeTotal
	if err := json.Unmarshal(body, &timeTotal); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return &timeTotal, nil
}

func (t *TeamworkAPI) GetLoggedTimeFromCalendarAPI(month, year int) (*LoggedTimeResponse, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	userID := strconv.Itoa(t.Config.UserID)
	baseURL := t.Config.ApiHost
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	url := fmt.Sprintf("%s/people/%s/loggedtime.json?m=%d&y=%d&projectId=0&page=1&pageSize=100",
		baseURL, userID, month, year)

	t.logDebug("Obtendo dados de tempo do endpoint de calendário: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	var auth string
	if strings.Contains(t.Config.AuthToken, ":") {
		auth = base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken))
	} else {
		auth = base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
	}
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TeamworkGoClient/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	if resp.StatusCode != 200 {
		t.logDebug("Resposta completa: %s", string(body))
		return nil, fmt.Errorf("erro ao obter dados de tempo (status %d): %s",
			resp.StatusCode, resp.Status)
	}

	var response LoggedTimeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta JSON: %v\nBody: %s", err, string(body))
	}

	if response.STATUS != "OK" {
		return nil, fmt.Errorf("resposta da API não está OK: %s", response.STATUS)
	}

	return &response, nil
}

func (t *TeamworkAPI) DownloadTimeReportPDF(startDate, endDate, filePath string) error {
	if !t.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("data inicial inválida: %v", err)
	}

	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("data final inválida: %v", err)
	}

	startDateFormatted := startTime.Format("2006-01-02T15:04:05+00:00")
	endDateFormatted := endTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second).Format("2006-01-02T15:04:05+00:00")

	baseURL := t.Config.ApiHost
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	params := url.Values{}
	params.Set("assignedTeamIds", "")
	params.Set("billableType", "all")
	params.Set("invoicedType", "all")
	params.Set("startDate", startDateFormatted)
	params.Set("endDate", endDateFormatted)
	params.Set("selectedColumns", "date,project,whoLoggedTime,descriptionAndTags,attachedTaskList,startTime,endTime,isEntryBillable,hasEntryBeenBilled,timeTaken,hoursTaken,estimatedTime,taskId")
	params.Set("onlyStarredProjects", "false")
	params.Set("includeArchivedProjects", "true")
	params.Set("matchAllTags", "true")
	params.Set("projectIds", "")
	params.Set("assignedToCompanyIds", "")
	params.Set("assignedToUserIds", strconv.Itoa(t.Config.UserID))
	params.Set("orderBy", "date")
	params.Set("orderMode", "desc")
	params.Set("projectStatuses", "all")
	params.Set("projectCompanyIds", "")

	downloadURL := fmt.Sprintf("%s/projects/api/v3/time.pdf?%s", baseURL, params.Encode())

	t.logDebug("Baixando relatório PDF de %s a %s...", startDate, endDate)
	t.logDebug("URL: %s", downloadURL)

	req, err := t.createRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro na requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao baixar relatório PDF: %d %s - %s",
			resp.StatusCode, resp.Status, string(bodyBytes[:minValue(len(bodyBytes), 200)]))
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório: %v", err)
		}
	}

	tempFile := filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo temporário: %v", err)
	}

	_, err = io.Copy(file, resp.Body)
	file.Close()

	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("erro ao salvar arquivo PDF: %v", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("erro ao mover arquivo para destino final: %v", err)
	}

	t.logDebug("Relatório PDF salvo em: %s", filePath)
	return nil
}

func (t *TeamworkAPI) DownloadCurrentMonthTimeReport() (string, error) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endDate := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())

	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	filePath, err := t.GetDefaultReportPath()
	if err != nil {
		return "", err
	}

	err = t.DownloadTimeReportPDF(startDateStr, endDateStr, filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}
