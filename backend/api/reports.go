package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
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

	fileName := fmt.Sprintf("TeamworkUtilization_%s.pdf", monthYear)
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

func formatMinutes(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d minutos", minutes)
	}

	hours := minutes / 60
	remainingMinutes := minutes % 60

	if remainingMinutes == 0 {
		return fmt.Sprintf("%d horas", hours)
	}

	return fmt.Sprintf("%d hora %d minutos", hours, remainingMinutes)
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}

	t, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		return timeStr
	}

	return t.Format("3:04pm")
}

func (t *TeamworkAPI) GenerateTimeReportTotalPDF(startDate, endDate, filePath string) error {
	if !t.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	entries, err := t.GetTimeEntriesForPeriod(startDate, endDate)
	if err != nil {
		return fmt.Errorf("erro ao obter entradas de tempo: %v", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("nenhuma entrada de tempo encontrada no período especificado")
	}

	totals, err := t.GetTimeTotalsForPeriod(startDate, endDate)
	if err != nil {
		return fmt.Errorf("erro ao obter totais de tempo: %v", err)
	}

	entriesByDate := make(map[string][]TimeEntryReport)
	var dates []string

	for _, entry := range entries {
		if _, exists := entriesByDate[entry.Date]; !exists {
			dates = append(dates, entry.Date)
		}
		entriesByDate[entry.Date] = append(entriesByDate[entry.Date], entry)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório: %v", err)
		}
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "Relatório do tempo total")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(190, 8, fmt.Sprintf("Date range: %s - %s", startDate, endDate))
	pdf.Ln(8)

	now := time.Now()
	userName := ""
	if len(entries) > 0 {
		userName = entries[0].UserFirstName + " " + entries[0].UserLastName
	}
	pdf.Cell(190, 8, fmt.Sprintf("Gerado para %s em %s", userName, now.Format("3:04pm 02/01/2006")))
	pdf.Ln(8)

	totalHours := float64(totals.TimeTotals.Minutes) / 60.0
	billableHours := float64(totals.TimeTotals.MinutesBillable) / 60.0
	pdf.Cell(190, 8, fmt.Sprintf("Totais filtrados: %.0f horas (%.2f) Contabilizável: %.0f horas (%.2f)",
		totalHours, totalHours, billableHours, billableHours))
	pdf.Ln(15)

	for _, date := range dates {
		dateEntries := entriesByDate[date]
		if len(dateEntries) == 0 {
			continue
		}

		parsedDate, _ := time.Parse("2006-01-02", date)
		displayDate := parsedDate.Format("Monday, 2 Jan 2006")

		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(190, 10, displayDate)
		pdf.Ln(10)

		pdf.SetFillColor(240, 240, 240)
		pdf.SetFont("Arial", "B", 8)

		headerY := pdf.GetY()
		colWidths := []float64{40, 25, 45, 25, 15, 15, 25, 25, 25, 15}
		headerTexts := []string{"Projeto", "Quem", "Descrição", "Lista de tarefas", "Início", "Fim", "Contabilizável", "Facturado", "Tempo", "Horas"}

		for i, width := range colWidths {
			pdf.SetX(10 + sumArray(colWidths, 0, i))
			pdf.SetFillColor(240, 240, 240)
			pdf.Cell(width, 8, headerTexts[i])
			x := 10 + sumArray(colWidths, 0, i)
			pdf.Rect(x, headerY, width, 8, "D")
		}
		pdf.Ln(8)

		pdf.SetFont("Arial", "", 8)

		dailyMinutes := 0
		dailyBillableMinutes := 0

		for _, entry := range dateEntries {
			rowY := pdf.GetY()

			pdf.SetX(10)
			pdf.Cell(colWidths[0], 8, entry.ProjectName)
			pdf.Rect(10, rowY, colWidths[0], 8, "D")

			pdf.SetX(10 + colWidths[0])
			pdf.Cell(colWidths[1], 8, fmt.Sprintf("%s %s", entry.UserFirstName, entry.UserLastName))
			pdf.Rect(10+colWidths[0], rowY, colWidths[1], 8, "D")

			description := entry.Description
			if entry.TaskName != "" {
				description = fmt.Sprintf("Tarefa: %s\n%s", entry.TaskName, entry.Description)
			}
			pdf.SetX(10 + colWidths[0] + colWidths[1])
			pdf.Cell(colWidths[2], 8, truncateText(description, 20))
			pdf.Rect(10+colWidths[0]+colWidths[1], rowY, colWidths[2], 8, "D")

			pdf.SetX(10 + colWidths[0] + colWidths[1] + colWidths[2])
			pdf.Cell(colWidths[3], 8, truncateText(entry.TasklistName, 10))
			pdf.Rect(10+colWidths[0]+colWidths[1]+colWidths[2], rowY, colWidths[3], 8, "D")

			pdf.SetX(10 + sumArray(colWidths, 0, 4))
			pdf.Cell(colWidths[4], 8, formatTime(entry.StartTime))
			pdf.Rect(10+sumArray(colWidths, 0, 4), rowY, colWidths[4], 8, "D")

			pdf.SetX(10 + sumArray(colWidths, 0, 5))
			pdf.Cell(colWidths[5], 8, formatTime(entry.EndTime))
			pdf.Rect(10+sumArray(colWidths, 0, 5), rowY, colWidths[5], 8, "D")

			billable := "Não"
			if entry.IsBillable {
				billable = "Sim"
				dailyBillableMinutes += entry.Minutes
			}
			pdf.SetX(10 + sumArray(colWidths, 0, 6))
			pdf.Cell(colWidths[6], 8, billable)
			pdf.Rect(10+sumArray(colWidths, 0, 6), rowY, colWidths[6], 8, "D")

			billed := "Não"
			if entry.IsBilled {
				billed = "Sim"
			}
			pdf.SetX(10 + sumArray(colWidths, 0, 7))
			pdf.Cell(colWidths[7], 8, billed)
			pdf.Rect(10+sumArray(colWidths, 0, 7), rowY, colWidths[7], 8, "D")

			timeStr := formatMinutes(entry.Minutes)
			pdf.SetX(10 + sumArray(colWidths, 0, 8))
			pdf.Cell(colWidths[8], 8, timeStr)
			pdf.Rect(10+sumArray(colWidths, 0, 8), rowY, colWidths[8], 8, "D")

			hours := float64(entry.Minutes) / 60.0
			pdf.SetX(10 + sumArray(colWidths, 0, 9))
			pdf.Cell(colWidths[9], 8, fmt.Sprintf("%.2f", hours))
			pdf.Rect(10+sumArray(colWidths, 0, 9), rowY, colWidths[9], 8, "D")

			pdf.Ln(8)
			dailyMinutes += entry.Minutes
		}

		totalRowY := pdf.GetY()
		pdf.SetFillColor(220, 220, 220)
		pdf.SetFont("Arial", "B", 8)

		pdf.SetX(10)
		pdf.Cell(190, 8, "Totais")
		pdf.Rect(10, totalRowY, 190, 8, "D")
		pdf.Ln(8)

		row1Y := pdf.GetY()
		pdf.SetX(10)
		pdf.Cell(175, 8, "Total")
		pdf.Rect(10, row1Y, 175, 8, "D")

		pdf.SetX(185)
		pdf.Cell(15, 8, fmt.Sprintf("%.2f", float64(dailyMinutes)/60.0))
		pdf.Rect(185, row1Y, 15, 8, "D")
		pdf.Ln(8)

		row2Y := pdf.GetY()
		pdf.SetX(10)
		pdf.Cell(175, 8, "Tempo Contabilizado")
		pdf.Rect(10, row2Y, 175, 8, "D")

		pdf.SetX(185)
		pdf.Cell(15, 8, fmt.Sprintf("%.2f", float64(dailyBillableMinutes)/60.0))
		pdf.Rect(185, row2Y, 15, 8, "D")
		pdf.Ln(10)
	}

	tempFile := filePath + ".tmp"
	err = pdf.OutputFileAndClose(tempFile)
	if err != nil {
		return fmt.Errorf("erro ao gerar PDF: %v", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("erro ao mover arquivo para destino final: %v", err)
	}

	t.logDebug("Relatório de tempo total salvo em: %s", filePath)
	return nil
}

func sumArray(array []float64, start, end int) float64 {
	var sum float64 = 0
	for i := start; i < end; i++ {
		sum += array[i]
	}
	return sum
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

func (t *TeamworkAPI) VerifyTimeExistsForPeriod(startDate, endDate string) (bool, error) {
	path := fmt.Sprintf("/projects/api/v3/time/total.json?startDate=%s&endDate=%s&userId=%d",
		startDate, endDate, t.Config.UserID)
	url := t.buildURL(path)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("erro ao verificar totais de tempo: %d %s", resp.StatusCode, resp.Status)
	}

	var timeTotal TimeTotal
	if err := json.Unmarshal(body, &timeTotal); err != nil {
		return false, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return timeTotal.TimeTotals.Minutes > 0, nil
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

func (t *TeamworkAPI) GenerateTimeReportFromCalendarAPI(month, year int, filePath string) error {
	response, err := t.GetLoggedTimeFromCalendarAPI(month, year)
	if err != nil {
		return fmt.Errorf("erro ao obter dados de tempo do calendário: %v", err)
	}

	if len(response.User.Billable) == 0 && len(response.User.Nonbillable) == 0 {
		return fmt.Errorf("nenhuma hora registrada encontrada para o período")
	}

	type DayEntry struct {
		Date               string
		BillableHours      float64
		NonBillableHours   float64
		TotalMinutes       int
		BillableMinutes    int
		NonBillableMinutes int
	}

	dateEntries := make(map[string]DayEntry)
	totalMinutes := 0
	totalBillableMinutes := 0
	totalNonBillableMinutes := 0

	// Processar entradas billable
	for _, entry := range response.User.Billable {
		if len(entry) < 3 {
			continue
		}

		timestamp, err := strconv.ParseInt(entry[0], 10, 64)
		if err != nil {
			continue
		}
		date := time.Unix(timestamp/1000, 0).Format("2006-01-02")

		hours, _ := strconv.ParseFloat(entry[1], 64)
		minutes, _ := strconv.Atoi(entry[2])

		existingEntry, exists := dateEntries[date]
		if !exists {
			existingEntry = DayEntry{
				Date: date,
			}
		}

		existingEntry.BillableHours += hours
		existingEntry.BillableMinutes += minutes
		existingEntry.TotalMinutes += minutes

		dateEntries[date] = existingEntry

		totalMinutes += minutes
		totalBillableMinutes += minutes
	}

	// Processar entradas não-billable
	for _, entry := range response.User.Nonbillable {
		if len(entry) < 3 {
			continue
		}

		timestamp, err := strconv.ParseInt(entry[0], 10, 64)
		if err != nil {
			continue
		}
		date := time.Unix(timestamp/1000, 0).Format("2006-01-02")

		hours, _ := strconv.ParseFloat(entry[1], 64)
		minutes, _ := strconv.Atoi(entry[2])

		existingEntry, exists := dateEntries[date]
		if !exists {
			existingEntry = DayEntry{
				Date: date,
			}
		}

		existingEntry.NonBillableHours += hours
		existingEntry.NonBillableMinutes += minutes
		existingEntry.TotalMinutes += minutes

		dateEntries[date] = existingEntry

		totalMinutes += minutes
		totalNonBillableMinutes += minutes
	}

	var dates []string
	for date := range dateEntries {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Inicialização do PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// Definir cores
	primaryColor := []int{94, 114, 228}      // #5E72E4 - Azul principal
	secondaryColor := []int{17, 205, 239}    // #11CDEF - Ciano
	accentColor := []int{45, 206, 137}       // #2DCE89 - Verde
	warningColor := []int{251, 99, 64}       // #FB6340 - Laranja
	neutralColor := []int{82, 95, 127}       // #525F7F - Azul neutro
	lightBgColor := []int{247, 250, 252}     // #F7FAFC - Fundo claro
	tableBorderColor := []int{233, 236, 239} // #E9ECEF
	tableHeaderColor := []int{246, 249, 252} // #F6F9FC
	tableStripeColor := []int{250, 251, 254} // #FAFBFE

	// ===== CABEÇALHO =====
	// Barra colorida no topo
	pdf.SetFillColor(primaryColor[0], primaryColor[1], primaryColor[2])
	pdf.Rect(0, 0, 210, 15, "F")

	// Título da aplicação
	pdf.SetFont("Arial", "B", 22)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetY(5)
	pdf.Cell(190, 8, "TEAMWORK LOGGER")

	// Subtítulo do relatório
	pdf.SetY(20)
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(primaryColor[0], primaryColor[1], primaryColor[2])
	pdf.Cell(190, 10, "Relatório de Horas Trabalhadas")

	// Box de informações
	pdf.SetY(35)
	pdf.SetFillColor(lightBgColor[0], lightBgColor[1], lightBgColor[2])
	pdf.Rect(15, 35, 180, 30, "F")

	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
	pdf.SetY(40)

	monthName := time.Month(month).String()
	pdf.SetX(20)
	pdf.Cell(50, 6, "Período:")
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(120, 6, fmt.Sprintf("%s %d", monthName, year))
	pdf.Ln(8)

	pdf.SetX(20)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(50, 6, "Colaborador:")
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(120, 6, fmt.Sprintf("%s %s", response.User.Firstname, response.User.Lastname))
	pdf.Ln(8)

	now := time.Now()
	pdf.SetX(20)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(50, 6, "Gerado em:")
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(120, 6, now.Format("02/01/2006 15:04:05"))

	// ===== RESUMO =====
	// Seção de resumo com cartões coloridos
	pdf.SetY(75)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
	pdf.Cell(190, 10, "Resumo de Horas")

	// Calcular valores para os cards
	totalHours := float64(totalMinutes) / 60.0
	billableHours := float64(totalBillableMinutes) / 60.0
	nonBillableHours := float64(totalNonBillableMinutes) / 60.0

	// Cartões de resumo
	cardWidth := 58.0
	cardHeight := 50.0
	cardSpacing := 3.0
	startY := 90.0

	// Cartão Total
	pdf.SetFillColor(primaryColor[0], primaryColor[1], primaryColor[2])
	pdf.Rect(15, startY, cardWidth, cardHeight, "F")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(20, startY+5)
	pdf.Cell(cardWidth-10, 6, "TOTAL")

	pdf.SetFont("Arial", "B", 16)
	pdf.SetXY(20, startY+15)
	pdf.Cell(cardWidth-10, 10, fmt.Sprintf("%.2f h", totalHours))

	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(20, startY+28)
	pdf.Cell(cardWidth-10, 5, fmt.Sprintf("%d minutos", totalMinutes))

	// Cartão Contabilizável
	secondCardX := 15 + cardWidth + cardSpacing
	pdf.SetFillColor(accentColor[0], accentColor[1], accentColor[2])
	pdf.Rect(secondCardX, startY, cardWidth, cardHeight, "F")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(secondCardX+5, startY+5)
	pdf.Cell(cardWidth-10, 6, "CONTABILIZÁVEL")

	pdf.SetFont("Arial", "B", 16)
	pdf.SetXY(secondCardX+5, startY+15)
	pdf.Cell(cardWidth-10, 10, fmt.Sprintf("%.2f h", billableHours))

	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(secondCardX+5, startY+28)
	pdf.Cell(cardWidth-10, 5, fmt.Sprintf("%d minutos", totalBillableMinutes))

	// Cartão Não Contabilizável
	thirdCardX := secondCardX + cardWidth + cardSpacing
	pdf.SetFillColor(secondaryColor[0], secondaryColor[1], secondaryColor[2])
	pdf.Rect(thirdCardX, startY, cardWidth, cardHeight, "F")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(thirdCardX+5, startY+5)
	pdf.Cell(cardWidth-10, 6, "NÃO CONTAB.")

	pdf.SetFont("Arial", "B", 16)
	pdf.SetXY(thirdCardX+5, startY+15)
	pdf.Cell(cardWidth-10, 10, fmt.Sprintf("%.2f h", nonBillableHours))

	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(thirdCardX+5, startY+28)
	pdf.Cell(cardWidth-10, 5, fmt.Sprintf("%d minutos", totalNonBillableMinutes))

	// Barra de progresso
	targetHours := 160.0 // Meta mensal padrão (8h/dia * ~20 dias úteis)
	progressPercentage := math.Min(100, (totalHours/targetHours)*100)

	barY := startY + cardHeight + 10
	pdf.SetY(barY)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
	pdf.Cell(190, 8, fmt.Sprintf("Progresso: %.1f%% da meta (%.1f/%.1f horas)", progressPercentage, totalHours, targetHours))

	barWidth := 180.0
	barHeight := 8.0
	fillWidth := (progressPercentage / 100.0) * barWidth

	// Fundo da barra
	pdf.SetFillColor(tableBorderColor[0], tableBorderColor[1], tableBorderColor[2])
	pdf.Rect(15, barY+8, barWidth, barHeight, "F")

	// Cor baseada no progresso
	barColor := primaryColor
	if progressPercentage < 50 {
		barColor = warningColor
	} else if progressPercentage >= 100 {
		barColor = accentColor
	}

	// Preenchimento da barra
	pdf.SetFillColor(barColor[0], barColor[1], barColor[2])
	if fillWidth > 0 {
		pdf.Rect(15, barY+8, fillWidth, barHeight, "F")
	}

	// ===== TABELA DE DETALHAMENTO =====
	detailY := barY + barHeight + 20
	pdf.SetY(detailY)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
	pdf.Cell(190, 10, "Detalhamento Diário")

	// Tabela de detalhamento
	tableY := detailY + 12
	pdf.SetY(tableY)

	// Largura das colunas
	colWidths := []float64{35, 40, 40, 35, 30}
	colHeaders := []string{"Data", "Contabilizável", "Não Contabilizável", "Total (h)", "Minutos"}

	// Cabeçalho da tabela
	pdf.SetFillColor(tableHeaderColor[0], tableHeaderColor[1], tableHeaderColor[2])
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])

	// Desenhar cabeçalho
	currentX := 15.0
	for i, header := range colHeaders {
		pdf.Rect(currentX, tableY, colWidths[i], 10, "F")
		pdf.SetXY(currentX, tableY+2)
		pdf.Cell(colWidths[i], 6, header)
		currentX += colWidths[i]
	}
	pdf.Ln(10)

	// Ordenar as datas em ordem inversa (mais recente primeiro)
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Linhas da tabela
	rowY := tableY + 10
	for i, date := range dates {
		entry := dateEntries[date]
		parsedDate, _ := time.Parse("2006-01-02", date)
		displayDate := parsedDate.Format("02/01/2006")

		// Verificar se é um dia da semana
		weekday := parsedDate.Weekday()
		isWeekend := weekday == time.Saturday || weekday == time.Sunday

		// Alternar cores das linhas
		fill := i%2 == 0
		if fill {
			pdf.SetFillColor(tableStripeColor[0], tableStripeColor[1], tableStripeColor[2])
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		// Destacar fins de semana
		if isWeekend {
			pdf.SetFillColor(252, 248, 227) // Cor amarelada para fins de semana
		}

		// Definir cor para as horas
		totalRowHours := float64(entry.TotalMinutes) / 60.0

		// Linha da tabela
		currentX := 15.0

		// Célula de data
		pdf.Rect(currentX, rowY, colWidths[0], 8, "F")
		pdf.SetXY(currentX, rowY+1)
		pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(colWidths[0], 6, displayDate)
		currentX += colWidths[0]

		// Célula de horas contabilizáveis
		pdf.Rect(currentX, rowY, colWidths[1], 8, "F")
		pdf.SetXY(currentX, rowY+1)

		// Verificar se é dia útil com menos de 8h
		if !isWeekend && totalRowHours < 8.0 && totalRowHours > 0 {
			pdf.SetTextColor(warningColor[0], warningColor[1], warningColor[2])
		} else {
			pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
		}

		pdf.Cell(colWidths[1], 6, fmt.Sprintf("%.2f", entry.BillableHours))
		currentX += colWidths[1]

		// Célula de horas não contabilizáveis
		pdf.Rect(currentX, rowY, colWidths[2], 8, "F")
		pdf.SetXY(currentX, rowY+1)
		pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
		pdf.Cell(colWidths[2], 6, fmt.Sprintf("%.2f", entry.NonBillableHours))
		currentX += colWidths[2]

		// Célula de total de horas
		pdf.Rect(currentX, rowY, colWidths[3], 8, "F")
		pdf.SetXY(currentX, rowY+1)

		// Destacar dias com menos de 8h
		if !isWeekend && totalRowHours < 8.0 && totalRowHours > 0 {
			pdf.SetTextColor(warningColor[0], warningColor[1], warningColor[2])
			pdf.SetFont("Arial", "B", 10)
		} else {
			pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
		}

		pdf.Cell(colWidths[3], 6, fmt.Sprintf("%.2f", totalRowHours))
		currentX += colWidths[3]

		// Célula de minutos
		pdf.Rect(currentX, rowY, colWidths[4], 8, "F")
		pdf.SetXY(currentX, rowY+1)
		pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(colWidths[4], 6, fmt.Sprintf("%d", entry.TotalMinutes))

		rowY += 8
	}

	// Linha de total
	pdf.SetFillColor(primaryColor[0], primaryColor[1], primaryColor[2])
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 10)

	currentX = 15.0

	// Total - Data
	pdf.Rect(currentX, rowY, colWidths[0], 10, "F")
	pdf.SetXY(currentX+2, rowY+2)
	pdf.Cell(colWidths[0]-4, 6, "TOTAL")
	currentX += colWidths[0]

	// Total - Contabilizável
	pdf.Rect(currentX, rowY, colWidths[1], 10, "F")
	pdf.SetXY(currentX, rowY+2)
	pdf.Cell(colWidths[1], 6, fmt.Sprintf("%.2f", billableHours))
	currentX += colWidths[1]

	// Total - Não Contabilizável
	pdf.Rect(currentX, rowY, colWidths[2], 10, "F")
	pdf.SetXY(currentX, rowY+2)
	pdf.Cell(colWidths[2], 6, fmt.Sprintf("%.2f", nonBillableHours))
	currentX += colWidths[2]

	// Total - Horas
	pdf.Rect(currentX, rowY, colWidths[3], 10, "F")
	pdf.SetXY(currentX, rowY+2)
	pdf.Cell(colWidths[3], 6, fmt.Sprintf("%.2f", totalHours))
	currentX += colWidths[3]

	// Total - Minutos
	pdf.Rect(currentX, rowY, colWidths[4], 10, "F")
	pdf.SetXY(currentX, rowY+2)
	pdf.Cell(colWidths[4], 6, fmt.Sprintf("%d", totalMinutes))

	// ===== RODAPÉ =====
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(neutralColor[0], neutralColor[1], neutralColor[2])
	pdf.MultiCell(180, 4, "Este relatório foi gerado automaticamente pelo Teamwork Logger. As horas contabilizáveis são aquelas marcadas como 'billable' no Teamwork.", "", "C", false)

	// Desenhar rodapé colorido
	pdf.SetFillColor(primaryColor[0], primaryColor[1], primaryColor[2])
	pdf.Rect(0, 297-10, 210, 10, "F")
	pdf.SetY(297 - 7)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "", 8)
	pdf.Cell(190, 4, fmt.Sprintf("Teamwork Logger • Relatório gerado em %s", now.Format("02/01/2006")))

	// Salvar o PDF
	tempFile := filePath + ".tmp"
	err = pdf.OutputFileAndClose(tempFile)
	if err != nil {
		return fmt.Errorf("erro ao gerar PDF: %v", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("erro ao mover arquivo para destino final: %v", err)
	}

	t.logDebug("Relatório da API de calendário salvo em: %s", filePath)
	return nil
}

func (t *TeamworkAPI) DownloadTimeReport(startDate, endDate, filePath string) error {
	if !t.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	hasTime, err := t.VerifyTimeExistsForPeriod(startDate, endDate)
	if err != nil {
		return fmt.Errorf("erro ao verificar se há tempo registrado: %v", err)
	}

	if !hasTime {
		return fmt.Errorf("nenhuma entrada de tempo encontrada no período especificado")
	}

	t.logDebug("Gerando relatório de tempo total a partir dos dados da API...")
	err = t.GenerateTimeReportTotalPDF(startDate, endDate, filePath)
	if err != nil {
		t.logDebug("Falha ao gerar relatório: %v", err)
		return err
	}

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

	filePath = strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_time_report.pdf"

	// Primeiro tentar com o método original
	t.logDebug("Tentando método original para gerar relatório...")
	err1 := t.DownloadTimeReport(startDateStr, endDateStr, filePath)
	if err1 == nil {
		return filePath, nil
	}
	t.logDebug("Método original falhou: %v", err1)

	// Se falhar, tentar com o endpoint alternativo
	t.logDebug("Tentando método alternativo com API de calendário...")
	err2 := t.GenerateTimeReportFromCalendarAPI(int(now.Month()), now.Year(), filePath)
	if err2 == nil {
		return filePath, nil
	}
	t.logDebug("Método alternativo falhou: %v", err2)

	// Se ambos falharem, retornar erro detalhado
	return "", fmt.Errorf("ambos os métodos falharam: %v; %v", err1, err2)
}
