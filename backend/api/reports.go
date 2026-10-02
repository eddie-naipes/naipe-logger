package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// maxTimeEntryPages limita a paginação de entradas de tempo. Com pageSize=500
// isso cobre 25 mil entradas — muito acima de qualquer período real de um
// usuário — e serve como rede de segurança contra laço infinito.
const maxTimeEntryPages = 50

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

// FlexString aceita texto ou número no JSON e guarda como texto. O
// loggedtime.json manda cada dia como ["1790812800000", 0.25, 15]: o
// timestamp vem como texto, horas e minutos como números. Tipado como
// [3]string, todo mês com horas lançadas falhava ao decodificar.
type FlexString string

func (f *FlexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("valor não é texto nem número: %s", data)
	}
	*f = FlexString(n.String())
	return nil
}

type LoggedTimeResponse struct {
	STATUS string `json:"STATUS"`
	User   struct {
		Billable    [][3]FlexString `json:"billable"`
		Firstname   string          `json:"firstname"`
		Lastname    string          `json:"lastname"`
		Nonbillable [][3]FlexString `json:"nonbillable"`
		ID          string          `json:"id"`
		Endepoch    string          `json:"endepoch"`
		Startepoch  string          `json:"startepoch"`
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

// v3Timelog é o formato de um lançamento em /projects/api/v3/time.json, que
// responde na chave "timelogs" (e não "timeEntries", como o código assumia).
type v3Timelog struct {
	ID          int    `json:"id"`
	Minutes     int    `json:"minutes"`
	Date        string `json:"date"`
	TimeLogged  string `json:"timeLogged"`
	Description string `json:"description"`
	IsBillable  bool   `json:"isBillable"`
	IsBilled    bool   `json:"isBilled"`
	TaskID      int    `json:"taskId"`
	ProjectID   int    `json:"projectId"`
	UserID      int    `json:"userId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (l v3Timelog) toReport() TimeEntryReport {
	date := l.Date
	if date == "" {
		date = l.TimeLogged
	}
	return TimeEntryReport{
		ID:          l.ID,
		ProjectID:   l.ProjectID,
		TaskID:      l.TaskID,
		UserID:      l.UserID,
		Date:        date,
		Hours:       float64(l.Minutes) / 60.0,
		Minutes:     l.Minutes,
		Description: l.Description,
		IsBillable:  l.IsBillable,
		IsBilled:    l.IsBilled,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
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

	// A v3 filtra quem lançou por assignedToUserIds (o mesmo filtro do PDF);
	// userId não é parâmetro documentado de time.json e fica só por
	// compatibilidade.
	path := fmt.Sprintf("/projects/api/v3/time.json?startDate=%s&endDate=%s&userId=%d&assignedToUserIds=%d",
		startDate, endDate, t.Config.UserID, t.Config.UserID)

	slog.Debug("Obtendo entradas de tempo", "inicio", startDate, "fim", endDate)

	allEntries := make([]TimeEntryReport, 0)
	err = t.fetchPages(t.buildURL(path), timeEntryPageSize, maxTimeEntryPages, "entradas de tempo",
		func(body []byte) (pageInfo, error) {
			// Variável nova a cada página: reaproveitar a mesma fazia o
			// Unmarshal sobrescrever o array da página anterior.
			var page struct {
				TimeEntries []TimeEntryReport `json:"timeEntries"`
				Timelogs    []v3Timelog       `json:"timelogs"`
				pageMeta
			}
			if err := json.Unmarshal(body, &page); err != nil {
				return pageInfo{}, err
			}

			for _, entry := range page.TimeEntries {
				entry.Date = normalizeEntryDate(entry.Date)
				allEntries = append(allEntries, entry)
			}
			for _, log := range page.Timelogs {
				entry := log.toReport()
				entry.Date = normalizeEntryDate(entry.Date)
				allEntries = append(allEntries, entry)
			}

			return pageInfo{
				items:   len(page.TimeEntries) + len(page.Timelogs),
				hasMore: page.Meta.Page.HasMore,
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return allEntries, nil
}

// normalizeEntryDate reduz a data do lançamento a YYYY-MM-DD, formato que o
// calendário e a detecção de conflitos comparam como texto. Valor
// irreconhecível é mantido como veio em vez de virar "0001-01-01".
func normalizeEntryDate(value string) string {
	if parsed, ok := parseTeamworkDate(value); ok {
		return parsed.Format("2006-01-02")
	}
	return value
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

	slog.Debug("Obtendo totais de tempo", "inicio", startDate, "fim", endDate)

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
			resp.StatusCode, resp.Status, string(body[:min(len(body), 100)]))
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
	url := t.buildURL(fmt.Sprintf("/people/%s/loggedtime.json?m=%d&y=%d&projectId=0&page=1&pageSize=100",
		userID, month, year))

	slog.Debug("Obtendo dados de tempo do endpoint de calendário", "url", url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TeamworkGoClient/1.0")

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		slog.Debug("Resposta do calendário", "status", resp.StatusCode, "corpo", sanitizeForLog(truncateForError(body, 300)))
		return nil, fmt.Errorf("erro ao obter dados de tempo (status %d): %s",
			resp.StatusCode, resp.Status)
	}

	var response LoggedTimeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		// Sem o corpo na mensagem: ele chega à tela e ao log e traz nome e
		// horas do usuário.
		slog.Debug("Resposta do calendário não decodificada", "corpo", sanitizeForLog(truncateForError(body, 300)))
		return nil, fmt.Errorf("erro ao decodificar resposta do calendário: %v", err)
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

	downloadURL := t.buildURL("/projects/api/v3/time.pdf?" + params.Encode())

	slog.Debug("Baixando relatório PDF", "inicio", startDate, "fim", endDate)

	req, err := t.createRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := getDownloadClient().Do(req)
	if err != nil {
		return fmt.Errorf("erro na requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao baixar relatório PDF: %d %s - %s",
			resp.StatusCode, resp.Status, string(bodyBytes[:min(len(bodyBytes), 200)]))
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

	slog.Info("Relatório PDF salvo", "arquivo", filePath)
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
