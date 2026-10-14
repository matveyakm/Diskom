package restController

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

// PaginationParams holds pagination query parameters
type PaginationParams struct {
	Limit  int
	Offset int
	Page   int
	Size   int
}

// ParsePaginationParams extracts pagination parameters from request
func ParsePaginationParams(r *http.Request) PaginationParams {
	params := PaginationParams{
		Limit:  20, // Default limit
		Offset: 0,  // Default offset
		Page:   1,  // Default page
		Size:   20, // Default size
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
			params.Size = limit
		}
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
			// Calculate page number from offset and limit
			if params.Limit > 0 {
				params.Page = (offset / params.Limit) + 1
			}
		}
	}

	// Parse page number (alternative to offset)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
			params.Offset = (page - 1) * params.Limit
		}
	}

	// Parse size (alternative to limit)
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			params.Size = size
			params.Limit = size
			// Recalculate offset if page is set
			if params.Page > 0 {
				params.Offset = (params.Page - 1) * params.Size
			}
		}
	}

	// Optional: Set maximum limit to prevent abuse
	const maxLimit = 100
	if params.Limit > maxLimit {
		params.Limit = maxLimit
		params.Size = maxLimit
	}

	return params
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination contains pagination metadata
type Pagination struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
	Offset      int   `json:"offset"`
	Limit       int   `json:"limit"`
}

// CRUDRepository handles basic CRUD operations for any GORM model
type CRUDRepository[T any] struct {
	db         *gorm.DB
	tableName  string
	httpClient *http.Client
}

// NewCRUDRepository creates a new repository instance
func NewCRUDRepository[T any](db *gorm.DB, tableName string) *CRUDRepository[T] {
	return &CRUDRepository[T]{
		db:        db,
		tableName: tableName,
		httpClient: &http.Client{
			Timeout: refRequestTimeout,
		},
	}
}

// HandleRequest routes HTTP requests to appropriate handlers
func (r *CRUDRepository[T]) HandleRequest(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		r.Create(w, req)
	case http.MethodGet:
		r.Get(w, req)
	case http.MethodPut:
		r.Update(w, req)
	case http.MethodDelete:
		r.Delete(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Create handles POST requests to create a new record
func (r *CRUDRepository[T]) Create(w http.ResponseWriter, req *http.Request) {
	var entity T
	if err := json.NewDecoder(req.Body).Decode(&entity); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Начинаем транзакцию
	tx := r.db.Begin()
	if tx.Error != nil {
		r.writeError(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	// Выполняем операцию в транзакции
	if err := tx.Table(r.tableName).Create(&entity).Error; err != nil {
		tx.Rollback() // Откат при ошибке
		r.writeError(w, "Failed to create record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Автогенерация "Кода" (см. dnumber.go): пустой dnumber + DNUMBER_PREFIX.
	if prefix := dnumberPrefix(); prefix != "" {
		if err := fillDnumber(tx, r.tableName, &entity, prefix); err != nil {
			tx.Rollback()
			r.writeError(w, "Failed to assign document number: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Если все успешно - коммитим
	if err := tx.Commit().Error; err != nil {
		r.writeError(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	r.writeJSONResponse(w, entity, http.StatusCreated)
}

// Get handles GET requests with pagination support
func (r *CRUDRepository[T]) Get(w http.ResponseWriter, req *http.Request) {
	// Check if this is a single record request or paginated list request
	id := req.URL.Query().Get("id")
	if id != "" {
		r.getByID(w, req, id)
		return
	}

	// Handle paginated list request
	r.getList(w, req)
}

// getByID retrieves a single record by ID
func (r *CRUDRepository[T]) getByID(w http.ResponseWriter, req *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		r.writeError(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	var entity T
	if err := r.db.Table(r.tableName).Where("id = ?", id).First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.writeError(w, "Record not found", http.StatusNotFound)
			return
		}
		r.writeError(w, "Failed to retrieve record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	r.writeJSONResponse(w, entity, http.StatusOK)
}

// getList retrieves a paginated list of records
func (r *CRUDRepository[T]) getList(w http.ResponseWriter, req *http.Request) {
	// Parse pagination parameters
	paginationParams := ParsePaginationParams(req)

	// Query builder
	query := r.db.Table(r.tableName)

	// Distinct column values for the checkbox filter list. Describes the
	// column, so row filters/pagination/sorting are ignored by design.
	params := req.URL.Query()
	if column, like, limit, ok, err := parseDistinct(params); err != nil {
		r.writeError(w, "Invalid distinct parameter: "+err.Error(), http.StatusBadRequest)
		return
	} else if ok {
		r.getDistinct(w, req.URL.Query(), column, like, limit)
		return
	}

	// Excel-style column filters: exact match + __suffix operators.
	// The legacy ?filter= flag is accepted but no longer required
	// (kept as a reserved word for old clients).
	filtered, ferr := applyFieldFilters(query, params)
	if ferr != nil {
		r.writeError(w, "Invalid filter parameter: "+ferr.Error(), http.StatusBadRequest)
		return
	}
	query = filtered

	// "Match nothing": any __in present but empty (all checkboxes cleared).
	if empty, ierr := hasEmptyIn(params); ierr != nil {
		r.writeError(w, "Invalid filter parameter: "+ierr.Error(), http.StatusBadRequest)
		return
	} else if empty {
		r.writeJSONResponse(w, PaginatedResponse{
			Data:       []T{},
			Pagination: Pagination{Limit: paginationParams.Limit, Offset: paginationParams.Offset},
		}, http.StatusOK)
		return
	}

	// Batch fetch by primary keys: ?ids=1,2,3 (see expand.go).
	if idsRaw := req.URL.Query().Get("ids"); idsRaw != "" {
		ids, err := parseIDs(idsRaw)
		if err != nil {
			r.writeError(w, "Invalid ids parameter: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(ids) == 0 {
			r.writeJSONResponse(w, PaginatedResponse{
				Data:       []T{},
				Pagination: Pagination{Limit: paginationParams.Limit, Offset: paginationParams.Offset},
			}, http.StatusOK)
			return
		}
		query = query.Where("id IN ?", ids)
	}

	// Global substring search across local columns (?q=&q_fields=).
	if fields, pattern, ok, serr := parseSearch(params); serr != nil {
		r.writeError(w, "Invalid search parameter: "+serr.Error(), http.StatusBadRequest)
		return
	} else if ok {
		query = applySearch(query, fields, pattern)
	}

	// Get total count for pagination metadata
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		r.writeError(w, "Failed to count records", http.StatusInternalServerError)
		return
	}

	// Ordering: validated ?sort=&order=, historical default keeps history.
	orderClause, oerr := parseSortOrder(params)
	if oerr != nil {
		r.writeError(w, "Invalid sort parameter: "+oerr.Error(), http.StatusBadRequest)
		return
	}

	// Apply pagination with proper ordering
	var entities []T
	err := query.
		Offset(paginationParams.Offset).
		Limit(paginationParams.Limit).
		Order(orderClause).
		Find(&entities).Error

	if err != nil {
		r.writeError(w, "Failed to retrieve records: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reference expansion: ?expand=id_department:department_directory (see expand.go).
	// Enriched rows carry "<column>_name"; the raw ids stay untouched.
	var data any = entities
	if expandRaw := req.URL.Query().Get("expand"); expandRaw != "" {
		specs, err := parseExpand(expandRaw)
		if err != nil {
			r.writeError(w, "Invalid expand parameter: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(specs) > 0 {
			raw, err := json.Marshal(entities)
			if err != nil {
				r.writeError(w, "Failed to encode records", http.StatusInternalServerError)
				return
			}
			var rows []map[string]any
			if err := json.Unmarshal(raw, &rows); err != nil {
				r.writeError(w, "Failed to encode records", http.StatusInternalServerError)
				return
			}
			enrichRows(r.httpClient, rows, specs)
			data = rows
		}
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(paginationParams.Limit) - 1) / int64(paginationParams.Limit))
	hasNext := paginationParams.Offset+paginationParams.Limit < int(totalCount)
	hasPrevious := paginationParams.Offset > 0

	// Build response
	response := PaginatedResponse{
		Data: data,
		Pagination: Pagination{
			CurrentPage: paginationParams.Page,
			PageSize:    len(entities),
			TotalPages:  totalPages,
			TotalCount:  totalCount,
			HasNext:     hasNext,
			HasPrevious: hasPrevious,
			Offset:      paginationParams.Offset,
			Limit:       paginationParams.Limit,
		},
	}

	r.writeJSONResponse(w, response, http.StatusOK)
}

// Update handles PUT requests to update an existing record
func (r *CRUDRepository[T]) Update(w http.ResponseWriter, req *http.Request) {
	var entity T
	if err := json.NewDecoder(req.Body).Decode(&entity); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Получаем ID из URL или из тела запроса
	id := req.URL.Query().Get("id")
	if id == "" {
		// Пробуем получить ID из структуры
		idValue := r.getIDFromEntity(entity)
		if idValue == nil {
			r.writeError(w, "Missing id parameter", http.StatusBadRequest)
			return
		}
		id = fmt.Sprintf("%v", idValue)
	}

	// Начинаем транзакцию
	tx := r.db.Begin()
	if tx.Error != nil {
		r.writeError(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	result := tx.Table(r.tableName).Where("id = ?", id).Updates(&entity)
	if result.Error != nil {
		tx.Rollback()
		r.writeError(w, "Failed to update record: "+result.Error.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		r.writeError(w, "Record not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit().Error; err != nil {
		r.writeError(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Получаем обновленную запись
	var updatedEntity T
	if err := r.db.Table(r.tableName).Where("id = ?", id).First(&updatedEntity).Error; err != nil {
		// Если не удалось получить, возвращаем то, что обновили
		r.writeJSONResponse(w, entity, http.StatusOK)
		return
	}
	r.writeJSONResponse(w, updatedEntity, http.StatusOK)
}

// Delete handles DELETE requests to remove a record
func (r *CRUDRepository[T]) Delete(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	if id == "" {
		r.writeError(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		r.writeError(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// Используем транзакцию для согласованности
	tx := r.db.Begin()
	if tx.Error != nil {
		r.writeError(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	result := tx.Table(r.tableName).Where("id = ?", idInt).Delete(nil)
	if result.Error != nil {
		tx.Rollback()
		r.writeError(w, "Failed to delete record: "+result.Error.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		r.writeError(w, "Record not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit().Error; err != nil {
		r.writeError(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getDistinct serves ?distinct=: capped distinct column values for the
// checkbox filter list (see filter.go). Sibling column filters and search
// narrow the list (so the second tab sees only what is left); the column's
// own keys are stripped (picked values must not hide the rest). NULLs are
// excluded (no blanks box in v1). Pagination/sorting are ignored by design.
func (r *CRUDRepository[T]) getDistinct(w http.ResponseWriter, params map[string][]string, column, like string, limit int) {
	sibling := stripOwnColumn(params, column)

	filtered := r.db.Table(r.tableName).Where(column + " IS NOT NULL")
	var err error
	if filtered, err = applyFieldFilters(filtered, sibling); err != nil {
		r.writeError(w, "Invalid filter parameter: "+err.Error(), http.StatusBadRequest)
		return
	}
	if fields, pattern, ok, serr := parseSearch(sibling); serr != nil {
		r.writeError(w, "Invalid search parameter: "+serr.Error(), http.StatusBadRequest)
		return
	} else if ok {
		filtered = applySearch(filtered, fields, pattern)
	}
	if empty, ierr := hasEmptyIn(sibling); ierr != nil {
		r.writeError(w, "Invalid filter parameter: "+ierr.Error(), http.StatusBadRequest)
		return
	} else if empty {
		r.writeJSONResponse(w, DistinctResponse{Data: []any{}, TotalDistinct: 0, Truncated: false}, http.StatusOK)
		return
	}
	if like != "" {
		filtered = filtered.Where("CAST("+column+" AS TEXT) ILIKE ?", "%"+escapeLike(like)+"%")
	}

	var rows []map[string]any
	valuesQuery := filtered.Session(&gorm.Session{}).
		Select("DISTINCT " + column).
		Order(column + " ASC").
		Limit(limit + 1)
	if err := valuesQuery.Find(&rows).Error; err != nil {
		r.writeError(w, "Failed to retrieve distinct values: "+err.Error(), http.StatusInternalServerError)
		return
	}
	truncated := len(rows) > limit
	if truncated {
		rows = rows[:limit]
	}
	vals := make([]any, 0, len(rows))
	for _, row := range rows {
		vals = append(vals, row[column])
	}

	var total int64
	countQuery := filtered.Session(&gorm.Session{}).Select("COUNT(DISTINCT " + column + ")")
	if err := countQuery.Scan(&total).Error; err != nil {
		r.writeError(w, "Failed to count distinct values: "+err.Error(), http.StatusInternalServerError)
		return
	}

	r.writeJSONResponse(w, DistinctResponse{
		Data:          vals,
		TotalDistinct: total,
		Truncated:     truncated,
	}, http.StatusOK)
}

// Helper function to write JSON response
func (r *CRUDRepository[T]) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Helper function to write error response
func (r *CRUDRepository[T]) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Helper function to extract ID from entity
func (r *CRUDRepository[T]) getIDFromEntity(entity T) interface{} {
	type Identifiable interface {
		GetID() int64
	}

	if identifiable, ok := any(entity).(Identifiable); ok {
		return identifiable.GetID()
	}
	return nil
}
