package restController

// Server-side filtering, sorting, searching and distinct values for list
// responses (Excel-style grid filters: popup per column, sort on header).
//
// Protocol (all on GET /api/<table>):
//
//	?sort=<col>&order=asc|desc        ordering, default "id DESC"
//	?q=<text>&q_fields=a,b            substring search across local columns
//	?<col>=<v>                        exact match (legacy ?filter= flag no longer needed)
//	?<col>__ne=<v>                    not equal
//	?<col>__gt=<v>  __gte __lt __lte  ordered comparison (PG coerces the
//	                                  literal to the column type, so numbers
//	                                  and ISO dates compare correctly)
//	?<col>__like=<frag>               substring (LIKE, special chars escaped)
//	?<col>__in=a,b,c                  set membership (checkboxes); present
//	                                  but empty means "match nothing" -> []
//	?distinct=<col>                   distinct column values for the checkbox
//	                                  list (capped, see below)
//	?distinct_like=<frag>             narrow the distinct list (substring)
//	?distinct_limit=<n>               cap override, default 200, max 500
//
// Design notes (same contract as expand.go):
//   - Our syntax is validated fail-closed (400): identifier shape, value
//     length caps, entry count caps, unknown order direction.
//   - Anything we cannot interpret is ignored, never executed: unknown
//     __suffix, invalid identifier, unknown query keys (fail-open on чужое).
//   - LIKE metacharacters (\ % _) in user input are escaped, otherwise a
//     user-typed "%" would match everything.
//   - ?distinct ignores row filters/pagination/sorting: it describes the
//     column, not the current page. NULLs are excluded (no blanks box in v1).
//   - The legacy ?filter= flag is accepted but no longer required and no
//     longer changes anything (kept as a reserved word for old clients).

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const (
	// maxFilterValueLen caps a single filter/search value.
	maxFilterValueLen = 200
	// maxSearchLen caps ?q=.
	maxSearchLen = 100
	// maxSearchFields caps ?q_fields= entries.
	maxSearchFields = 10
	// maxInValues caps ?field__in= entries (same as maxBatchIDs).
	maxInValues = 200
	// defaultDistinctLimit caps ?distinct= rows unless overridden.
	defaultDistinctLimit = 200
	// maxDistinctLimit caps ?distinct_limit=.
	maxDistinctLimit = 500
)

var (
	// Precompiled: hot path, same shape as reExpandColumn.
	reFilterColumn = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
)

// filterOp is a validated per-column predicate.
type filterOp struct {
	column string
	op     string // one of: eq ne gt gte lt lte like in
	value  string // for in: raw comma list (parsed later)
}

// knownSuffix maps "__suffix" to the canonical op name.
var knownSuffix = map[string]string{
	"__ne": "ne", "__gt": "gt", "__gte": "gte",
	"__lt": "lt", "__lte": "lte", "__like": "like", "__in": "in",
}

// reservedQueryKeys are never treated as column filters.
var reservedQueryKeys = map[string]struct{}{
	"id": {}, "limit": {}, "offset": {}, "page": {}, "size": {},
	"filter": {}, "ids": {}, "expand": {},
	"sort": {}, "order": {}, "q": {}, "q_fields": {},
	"distinct": {}, "distinct_like": {}, "distinct_limit": {},
}

// splitFilterKey interprets one query key. Returns ok=false for anything
// that must be ignored (reserved, unknown suffix, invalid identifier).
func splitFilterKey(key string) (op filterOp, ok bool) {
	if _, reserved := reservedQueryKeys[key]; reserved {
		return filterOp{}, false
	}
	column, suffix, hasSuffix := strings.Cut(key, "__")
	if hasSuffix {
		canonical, known := knownSuffix["__"+suffix]
		if !known || !reFilterColumn.MatchString(column) {
			return filterOp{}, false
		}
		return filterOp{column: column, op: canonical}, true
	}
	if !reFilterColumn.MatchString(key) {
		return filterOp{}, false
	}
	return filterOp{column: key, op: "eq"}, true
}

// escapeLike escapes LIKE metacharacters so user input is matched literally.
// Bound parameters are not subject to string-literal parsing, so backslash
// works as the LIKE escape character with no explicit ESCAPE clause needed.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// checkValueLen fail-closes overlong values (our syntax, our caps).
func checkValueLen(v string) error {
	if len(v) > maxFilterValueLen {
		return fmt.Errorf("filter value too long (max %d chars)", maxFilterValueLen)
	}
	return nil
}

// applyFieldFilters adds exact/operator predicates for recognized keys.
// Unknown keys are ignored. Overlong values are an error (400).
func applyFieldFilters(query *gorm.DB, params map[string][]string) (*gorm.DB, error) {
	for key, values := range params {
		op, ok := splitFilterKey(key)
		if !ok || len(values) == 0 {
			continue
		}
		raw := values[0]
		if raw == "" && op.op != "in" {
			continue
		}
		switch op.op {
		case "eq":
			if err := checkValueLen(raw); err != nil {
				return nil, err
			}
			query = query.Where(op.column+" = ?", raw)
		case "ne":
			if err := checkValueLen(raw); err != nil {
				return nil, err
			}
			query = query.Where(op.column+" <> ?", raw)
		case "gt", "gte", "lt", "lte":
			if err := checkValueLen(raw); err != nil {
				return nil, err
			}
			symbols := map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<="}
			query = query.Where(op.column+" "+symbols[op.op]+" ?", raw)
		case "like":
			if err := checkValueLen(raw); err != nil {
				return nil, err
			}
			query = query.Where("CAST("+op.column+" AS TEXT) ILIKE ?", "%"+escapeLike(raw)+"%")
		case "in":
			vals, _, err := parseInValues(raw)
			if err != nil {
				return nil, err
			}
			if len(vals) == 0 {
				continue // empty set handled by caller (match nothing)
			}
			query = query.Where(op.column+" IN ?", vals)
		}
	}
	return query, nil
}

// hasEmptyIn reports whether any __in key is present with zero values.
// Such a request means "match nothing" (all checkboxes cleared).
func hasEmptyIn(params map[string][]string) (bool, error) {
	for key, values := range params {
		op, ok := splitFilterKey(key)
		if !ok || op.op != "in" || len(values) == 0 {
			continue
		}
		vals, _, err := parseInValues(values[0])
		if err != nil {
			return false, err
		}
		if len(vals) == 0 {
			return true, nil
		}
	}
	return false, nil
}

// stripOwnColumn drops the distinct column's own keys from the params:
// selecting from this list must not narrow it (otherwise picked values
// would hide the rest). Everything else (sibling filters, search) stays.
func stripOwnColumn(params map[string][]string, column string) map[string][]string {
	out := make(map[string][]string, len(params))
	for key, values := range params {
		if op, ok := splitFilterKey(key); ok && op.column == column {
			continue
		}
		out[key] = values
	}
	return out
}

// countPredicates counts recognized filter keys with non-blank values
// (exact match and operator suffixes). Reserved words and
// uninterpretable keys don't count. Zero means "no filter": the caller
// (DELETE) refuses the request — mass-wipe guard.
func countPredicates(params map[string][]string) (int, error) {
	n := 0
	for key, values := range params {
		if _, ok := splitFilterKey(key); !ok {
			continue
		}
		for _, v := range values {
			if len(v) > maxFilterValueLen {
				return 0, fmt.Errorf("filter value too long (max %d chars)", maxFilterValueLen)
			}
			if strings.TrimSpace(v) != "" {
				n++
				break
			}
		}
	}
	return n, nil
}

// updateColumns lists JSON body keys to write on Update: everything
// present except the primary key and explicit nulls. Present-but-zero
// values (e.g. status 0) ARE written — struct-only Updates silently
// skipped them, which made any status→0 transition 404. Explicit nulls
// stay skipped (a null must never wipe a column, e.g. dnumber).
func updateColumns(body map[string]any) []string {
	cols := make([]string, 0, len(body))
	for key, value := range body {
		if key == "id" || value == nil {
			continue
		}
		cols = append(cols, key)
	}
	return cols
}

// parseInValues splits a comma list, trims, drops empties.
func parseInValues(raw string) ([]string, bool, error) {
	parts := strings.Split(raw, ",")
	vals := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) > maxFilterValueLen {
			return nil, false, fmt.Errorf("in value too long (max %d chars)", maxFilterValueLen)
		}
		vals = append(vals, p)
	}
	if len(vals) > maxInValues {
		return nil, false, fmt.Errorf("too many in values (max %d)", maxInValues)
	}
	return vals, len(vals) == 0, nil
}

// parseSortOrder validates ?sort=&order= into an ORDER BY clause.
// Empty sort keeps the historical default. Errors are 400.
func parseSortOrder(params map[string][]string) (string, error) {
	sort := firstParam(params, "sort")
	if strings.TrimSpace(sort) == "" {
		return "id DESC", nil
	}
	sort = strings.TrimSpace(sort)
	if !reFilterColumn.MatchString(sort) {
		return "", fmt.Errorf("invalid sort column %q", sort)
	}
	order := strings.ToLower(strings.TrimSpace(firstParam(params, "order")))
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		return "", fmt.Errorf("invalid order %q, want asc|desc", order)
	}
	return sort + " " + strings.ToUpper(order), nil
}

// parseSearch validates ?q=&q_fields= into fields + ILIKE pattern.
// ok=false when no search requested. Errors are 400.
func parseSearch(params map[string][]string) (fields []string, pattern string, ok bool, err error) {
	q := firstParam(params, "q")
	if q == "" {
		return nil, "", false, nil
	}
	if len(q) > maxSearchLen {
		return nil, "", false, fmt.Errorf("search text too long (max %d chars)", maxSearchLen)
	}
	rawFields := firstParam(params, "q_fields")
	if strings.TrimSpace(rawFields) == "" {
		return nil, "", false, fmt.Errorf("q_fields required with q")
	}
	for _, f := range strings.Split(rawFields, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !reFilterColumn.MatchString(f) {
			return nil, "", false, fmt.Errorf("invalid search field %q", f)
		}
		fields = append(fields, f)
	}
	if len(fields) == 0 {
		return nil, "", false, fmt.Errorf("q_fields required with q")
	}
	if len(fields) > maxSearchFields {
		return nil, "", false, fmt.Errorf("too many search fields (max %d)", maxSearchFields)
	}
	return fields, "%" + escapeLike(q) + "%", true, nil
}

// applySearch ORs one ILIKE per field inside explicit parentheses so it
// composes safely with the AND chain of column filters.
func applySearch(query *gorm.DB, fields []string, pattern string) *gorm.DB {
	conds := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))
	for _, f := range fields {
		conds = append(conds, "CAST("+f+" AS TEXT) ILIKE ?")
		args = append(args, pattern)
	}
	return query.Where("("+strings.Join(conds, " OR ")+")", args...)
}

// parseDistinct validates ?distinct=&distinct_like=&distinct_limit=.
// ok=false when no distinct requested. Errors are 400.
func parseDistinct(params map[string][]string) (column, like string, limit int, ok bool, err error) {
	column = strings.TrimSpace(firstParam(params, "distinct"))
	if column == "" {
		return "", "", 0, false, nil
	}
	if !reFilterColumn.MatchString(column) {
		return "", "", 0, false, fmt.Errorf("invalid distinct column %q", column)
	}
	like = firstParam(params, "distinct_like")
	if len(like) > maxSearchLen {
		return "", "", 0, false, fmt.Errorf("distinct_like too long (max %d chars)", maxSearchLen)
	}
	limit = defaultDistinctLimit
	if raw := strings.TrimSpace(firstParam(params, "distinct_limit")); raw != "" {
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || n <= 0 || n > maxDistinctLimit {
			return "", "", 0, false, fmt.Errorf("invalid distinct_limit %q (1..%d)", raw, maxDistinctLimit)
		}
		limit = n
	}
	return column, like, limit, true, nil
}

// DistinctResponse is the ?distinct= payload.
type DistinctResponse struct {
	Data          []any `json:"data"`
	TotalDistinct int64 `json:"total_distinct"`
	Truncated     bool  `json:"truncated"`
}

func firstParam(params map[string][]string, key string) string {
	if v, found := params[key]; found && len(v) > 0 {
		return v[0]
	}
	return ""
}
