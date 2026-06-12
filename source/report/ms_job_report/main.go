package main

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

//go:embed assets/fonts/ARIAL.TTF
var arialRegular []byte

//go:embed assets/fonts/ARIALBD.TTF
var arialBold []byte

type Config struct {
	HasPageNumber bool
}

type MrtService struct {
	config Config
	maroto core.Maroto
}

func New(cfg Config) *MrtService {
	marotoCfgBuilder := config.NewBuilder()

	marotoCfgBuilder.WithCustomFonts([]*entity.CustomFont{
		{Family: "Arial", Style: fontstyle.Normal, Bytes: arialRegular},
		{Family: "Arial", Style: fontstyle.Bold, Bytes: arialBold},
	})

	marotoCfgBuilder.WithDefaultFont(&props.Font{
		Family: "Arial",
		Style:  fontstyle.Normal,
		Size:   10,
	})

	if cfg.HasPageNumber {
		fmt.Println("v2 Service: Page numbers are enabled.")
		marotoCfgBuilder.WithPageNumber()
	}

	m := maroto.New(marotoCfgBuilder.Build())

	return &MrtService{
		config: cfg,
		maroto: m,
	}
}

type PDFRequest struct {
	Title     string `json:"title"`
	LeftText  string `json:"left_text"`
	RightText string `json:"right_text"`
}

// Object описывает структуру данных, получаемую из внешнего REST API
type Object struct {
	Id             int64     `gorm:"column:id;primaryKey" json:"id"`
	Ddate          time.Time `gorm:"column:ddate" json:"ddate"`
	Dnumber        string    `gorm:"column:dnumber" json:"dnumber"`
	Note           string    `gorm:"column:note" json:"note"`
	IdBasis        int64     `gorm:"column:id_basis" json:"id_basis"`
	Status         int16     `gorm:"column:status" json:"status"`
	IdUser         int64     `gorm:"column:id_user" json:"id_user"`
	Prefix         int16     `gorm:"column:prefix" json:"prefix"`
	IdEmployee     int64     `gorm:"column:id_employee" json:"id_employee"`
	IdOrganization int64     `gorm:"column:id_organization" json:"id_organization"`
	IdDepartment   int64     `gorm:"column:id_department" json:"id_department"`
	IdJobTitle     int64     `gorm:"column:id_job_title" json:"id_job_title"`
	Justification  string    `gorm:"column:justification" json:"justification"`
	IdTransaction  string    `gorm:"column:id_transaction" json:"id_transaction"`
}

type LedgerResponse struct {
	Data       []Object    `json:"data"`
	Pagination interface{} `json:"pagination"`
}

// DirectoryResponse универсальная структура для получения наименования/значения из каталогов.
// Измените json тег (например на "name" или "title"), если ваш API возвращает другое свойство.
type DirectoryResponse struct {
	Name string `json:"name"`
}

type Builder struct {
	cfg Config
}

func NewBuilder() *Builder { return &Builder{} }
func (b *Builder) WithPageNumber() *Builder {
	b.cfg.HasPageNumber = true
	return b
}
func (b *Builder) Build() Config { return b.cfg }

func main() {
	http.HandleFunc("/api/job_report", pdfHandler)

	log.Println("PDF microservice starting on port :8443...")
	if err := http.ListenAndServe(":8443", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func pdfHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("PDF Generator Service is running. Send POST request with JSON payload."))
		return
	}

	var req PDFRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON body format: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Missing required field: title", http.StatusUnprocessableEntity)
		return
	}

	apiURL := "http://172.25.0.37:8443/api/appointment_document"

	cfgHTTP := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: cfgHTTP, Timeout: 10 * time.Second}

	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch REST data: %v", err)
		http.Error(w, "Ledger API unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Ledger API returned unexpected status: %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response body: %v", err)
		http.Error(w, "Failed to read API response", http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] Raw Ledger JSON Response: %s", string(bodyBytes))
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var ledgerEnvelope LedgerResponse
	if err := json.NewDecoder(resp.Body).Decode(&ledgerEnvelope); err != nil {
		log.Printf("[ERROR] Failed to decode ledger JSON: %v", err)
		http.Error(w, "Malformed ledger data structure: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Инициализация карт для кэширования имен из справочников
	employees := make(map[int64]string)
	organizations := make(map[int64]string)
	departments := make(map[int64]string)
	jobTitles := make(map[int64]string)

	// Наполнение словарей значениями из внешних REST-справочников
	for _, obj := range ledgerEnvelope.Data {
		if _, exists := employees[obj.IdEmployee]; !exists {
			employees[obj.IdEmployee] = fetchDirectoryValue(client, "http://172.25.0.25:8443/api/employee_directory", obj.IdEmployee)
		}
		if _, exists := organizations[obj.IdOrganization]; !exists {
			organizations[obj.IdOrganization] = fetchDirectoryValue(client, "http://172.25.0.31:8443/api/organization_directory", obj.IdOrganization)
		}
		if _, exists := departments[obj.IdDepartment]; !exists {
			departments[obj.IdDepartment] = fetchDirectoryValue(client, "http://172.25.0.27:8443/api/department_directory", obj.IdDepartment)
		}
		if _, exists := jobTitles[obj.IdJobTitle]; !exists {
			jobTitles[obj.IdJobTitle] = fetchDirectoryValue(client, "http://172.25.0.35:8443/api/job_title_directory", obj.IdJobTitle)
		}
	}

	cfg := NewBuilder().WithPageNumber().Build()
	mrt := New(cfg)

	var rows []core.Row

	rows = append(rows, row.New(20).Add(
		col.New(12).Add(
			text.New(req.Title, props.Text{
				Top:   5,
				Size:  18,
				Style: fontstyle.Bold,
				Align: align.Center,
			}),
		),
	))

	rows = append(rows, row.New(10))

	headers := []string{"ID", "Дата", "Номер", "Сотрудник", "Организация", "Отдел", "Должность", "Основание"}

	headerRow := row.New(10)
	// Распределение сетки (Сумма = 12): 0.5 + 1.0 + 0.8 + 1.7 + 1.7 + 1.5 + 1.8 + 3.0 = 12 (переведено в целые числа умножением на масштаб, либо пропорциями)
	// Используем целые числа, оптимальные для текстовых названий: 1+1+1+2+2+1+1+3 = 12
	headerRow.Add(col.New(1).Add(text.New(headers[0], props.Text{Style: fontstyle.Bold, Size: 8})))
	headerRow.Add(col.New(1).Add(text.New(headers[1], props.Text{Style: fontstyle.Bold, Size: 8})))
	headerRow.Add(col.New(1).Add(text.New(headers[2], props.Text{Style: fontstyle.Bold, Size: 8})))
	headerRow.Add(col.New(2).Add(text.New(headers[3], props.Text{Style: fontstyle.Bold, Size: 8}))) // Больше места для ФИО
	headerRow.Add(col.New(2).Add(text.New(headers[4], props.Text{Style: fontstyle.Bold, Size: 8}))) // Больше места для Орг
	headerRow.Add(col.New(1).Add(text.New(headers[5], props.Text{Style: fontstyle.Bold, Size: 8})))
	headerRow.Add(col.New(1).Add(text.New(headers[6], props.Text{Style: fontstyle.Bold, Size: 8})))
	headerRow.Add(col.New(3).Add(text.New(headers[7], props.Text{Style: fontstyle.Bold, Size: 8})))
	rows = append(rows, headerRow)

	rows = append(rows, row.New(2))

	for _, obj := range ledgerEnvelope.Data {
		dataRow := row.New(12) // Высота увеличена до 12 для многострочных текстовых названий

		dateStr := obj.Ddate.Format("02.01.2006")

		// Извлекаем текстовые значения из кэшированных мап вместо ID
		empName := employees[obj.IdEmployee]
		orgName := organizations[obj.IdOrganization]
		deptName := departments[obj.IdDepartment]
		jobName := jobTitles[obj.IdJobTitle]

		dataRow.Add(col.New(1).Add(text.New(strconv.FormatInt(obj.Id, 10), props.Text{Size: 7})))
		dataRow.Add(col.New(1).Add(text.New(dateStr, props.Text{Size: 7})))
		dataRow.Add(col.New(1).Add(text.New(obj.Dnumber, props.Text{Size: 7})))
		dataRow.Add(col.New(2).Add(text.New(empName, props.Text{Size: 7})))
		dataRow.Add(col.New(2).Add(text.New(orgName, props.Text{Size: 7})))
		dataRow.Add(col.New(1).Add(text.New(deptName, props.Text{Size: 7})))
		dataRow.Add(col.New(1).Add(text.New(jobName, props.Text{Size: 7})))
		dataRow.Add(col.New(3).Add(text.New(obj.Justification, props.Text{Size: 7})))

		rows = append(rows, dataRow)
	}
	mrt.maroto.AddRows(rows...)

	document, err := mrt.maroto.Generate()
	if err != nil {
		log.Printf("[ERROR] Maroto PDF generation failed: %v", err)
		http.Error(w, "Failed to compile document architecture: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes := document.GetBytes()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"document.pdf\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))

	_, _ = w.Write(pdfBytes)
}

// Вспомогательная функция для отправки запросов к справочникам.
// Если запрос падает или ID равен 0, возвращает исходный строковый ID как запасной вариант.
// Вспомогательная функция для отправки запросов к справочникам.
// Теперь при id == 0 всё равно делается запрос, чтобы вернуть текстовое поле Name.
func fetchDirectoryValue(client *http.Client, baseURL string, id int64) string {
	// Конструируем URL вида: http://172.25.0
	url := fmt.Sprintf("%s?id=%d", baseURL, id)

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[WARN] Directory fetch failed for URL %s: %v", url, err)
		return strconv.FormatInt(id, 10)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return strconv.FormatInt(id, 10)
	}

	var dirResp DirectoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&dirResp); err != nil {
		log.Printf("[WARN] Directory JSON decode failed for URL %s: %v", url, err)
		return strconv.FormatInt(id, 10)
	}

	// Если сервер вернул пустое имя, откатываемся к строковому ID
	if dirResp.Name == "" {
		return strconv.FormatInt(id, 10)
	}

	return dirResp.Name
}
