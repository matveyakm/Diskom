package restController

// Reference expansion for list responses.
//
// Protocol:
//
//	GET /api/<table>?expand=<col>:<endpoint>,...
//	GET /api/<table>?ids=1,2,3
//
// expand enriches every returned row with "<col>_name" resolved from the
// referenced directory service (e.g. ?expand=id_department:department_directory
// adds "id_department_name"). ids fetches several records by primary key
// in one request (used by expand internally, also usable directly).
//
// Design notes:
//   - Reference addresses come ONLY from server-side env REF_<ENDPOINT>,
//     never from the client (no SSRF: the client controls just a validated
//     endpoint name, the base URL is operator-controlled and restricted
//     to http/https).
//   - Resolution is fail-open per spec: a broken/missing reference service
//     leaves raw ids in place, the list itself still returns 200.
//   - No N+1: distinct ids of one page are fetched with a single batched
//     request per referenced endpoint (parallel across endpoints).
//   - No caching: directories are read fresh on every call, so renames
//     are visible immediately. With page size <= 100 this is cheap.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxExpandSpecs caps ?expand= entries per request.
	maxExpandSpecs = 5
	// maxBatchIDs caps ?ids= entries per request.
	maxBatchIDs = 200
	// refRequestTimeout caps a single reference-service call.
	refRequestTimeout = 5 * time.Second
	// maxRefBodyBytes caps a reference-service response body.
	maxRefBodyBytes = 1 << 20 // 1 MiB
	// refDisplayField is the directory row field used as human-readable name.
	refDisplayField = "name"
	// refResultSuffix is appended to the source column: id_department -> id_department_name.
	refResultSuffix = "_name"
)

var (
	// Precompiled: hot path, no reason to recompile per request.
	reExpandColumn   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
	reExpandEndpoint = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	reDigits         = regexp.MustCompile(`^[0-9]+$`)
)

// expandSpec is one validated "<column>:<endpoint>" pair.
type expandSpec struct {
	column   string
	endpoint string
}

// parseExpand validates raw ?expand= value.
// Returns error on malformed syntax (fail-closed: caller must answer 400).
// A syntactically valid but unconfigured endpoint is NOT an error here;
// it is skipped later during resolution (fail-open).
func parseExpand(raw string) ([]expandSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > maxExpandSpecs {
		return nil, fmt.Errorf("too many expand specs (max %d)", maxExpandSpecs)
	}
	specs := make([]expandSpec, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		col, ep, ok := strings.Cut(p, ":")
		if !ok {
			return nil, fmt.Errorf("invalid expand spec %q, want <column>:<endpoint>", p)
		}
		col, ep = strings.TrimSpace(col), strings.TrimSpace(ep)
		if !reExpandColumn.MatchString(col) {
			return nil, fmt.Errorf("invalid expand column %q", col)
		}
		if !reExpandEndpoint.MatchString(ep) {
			return nil, fmt.Errorf("invalid expand endpoint %q", ep)
		}
		specs = append(specs, expandSpec{column: col, endpoint: ep})
	}
	return specs, nil
}

// parseIDs validates raw ?ids= value into distinct positive ids.
// Returns error on any malformed token or over the cap (fail-closed).
func parseIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > maxBatchIDs {
		return nil, fmt.Errorf("too many ids (max %d)", maxBatchIDs)
	}
	seen := make(map[int64]struct{}, len(parts))
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !reDigits.MatchString(p) {
			return nil, fmt.Errorf("invalid id %q", p)
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid id %q", p)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

// resolveRefBase returns the operator-configured base URL for an endpoint,
// e.g. endpoint "employee_directory" -> env REF_EMPLOYEE_DIRECTORY.
// Only http/https URLs are accepted.
func resolveRefBase(endpoint string) (string, bool) {
	raw := strings.TrimSpace(os.Getenv("REF_" + strings.ToUpper(endpoint)))
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		log.Printf("[expand] WARN: bad REF_%s value, skipping: %v", strings.ToUpper(endpoint), err)
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		log.Printf("[expand] WARN: REF_%s must be http(s), skipping", strings.ToUpper(endpoint))
		return "", false
	}
	if u.Host == "" {
		log.Printf("[expand] WARN: REF_%s has no host, skipping", strings.ToUpper(endpoint))
		return "", false
	}
	return strings.TrimSuffix(raw, "/"), true
}

// refIDKey normalizes a JSON-decoded id value into a canonical map key.
func refIDKey(v any) (string, bool) {
	switch t := v.(type) {
	case float64:
		if t != float64(int64(t)) || t <= 0 {
			return "", false
		}
		return strconv.FormatInt(int64(t), 10), true
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return "", false
		}
		if reDigits.MatchString(t) {
			if t = strings.TrimLeft(t, "0"); t == "" {
				return "", false // was all zeros
			}
		}
		return t, true
	default:
		return "", false
	}
}

// refDisplayName extracts the human-readable name from a directory row.
func refDisplayName(row map[string]any) (string, bool) {
	name, ok := row[refDisplayField].(string)
	if !ok {
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return name, true
}

// refListEnvelope is the minimal shape of a directory list response.
type refListEnvelope struct {
	Data []map[string]any `json:"data"`
}

// fetchRefMap loads id -> display name for one endpoint in a single request.
func fetchRefMap(client *http.Client, baseURL, endpoint string, ids []int64) (map[string]string, error) {
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.FormatInt(id, 10)
	}
	reqURL := baseURL + "/api/" + endpoint + "?ids=" + url.QueryEscape(strings.Join(strIDs, ",")) + "&limit=200"

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reference service %s returned %s", endpoint, resp.Status)
	}
	var env refListEnvelope
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRefBodyBytes)).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", endpoint, err)
	}
	out := make(map[string]string, len(env.Data))
	for _, row := range env.Data {
		key, ok := refIDKey(row["id"])
		if !ok {
			continue
		}
		if name, ok := refDisplayName(row); ok {
			out[key] = name
		}
	}
	return out, nil
}

// collectSpecIDs gathers distinct positive ids of one column across rows.
func collectSpecIDs(rows []map[string]any, column string) []int64 {
	seen := make(map[int64]struct{})
	var ids []int64
	for _, row := range rows {
		key, ok := refIDKey(row[column])
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// enrichRows resolves every spec against its directory service (in parallel)
// and writes "<column>_name" into matching rows. Never fails the request:
// any per-spec error only logs and leaves raw ids in place.
func enrichRows(client *http.Client, rows []map[string]any, specs []expandSpec) {
	var wg sync.WaitGroup
	for _, spec := range specs {
		ids := collectSpecIDs(rows, spec.column)
		if len(ids) == 0 {
			continue
		}
		base, ok := resolveRefBase(spec.endpoint)
		if !ok {
			log.Printf("[expand] INFO: no REF configured for %q, leaving raw ids", spec.endpoint)
			continue
		}
		wg.Add(1)
		go func(spec expandSpec, ids []int64, base string) {
			defer wg.Done()
			names, err := fetchRefMap(client, base, spec.endpoint, ids)
			if err != nil {
				log.Printf("[expand] WARN: %s: %v, leaving raw ids", spec.endpoint, err)
				return
			}
			for _, row := range rows {
				key, ok := refIDKey(row[spec.column])
				if !ok {
					continue
				}
				if name, ok := names[key]; ok {
					row[spec.column+refResultSuffix] = name
				}
			}
		}(spec, ids, base)
	}
	wg.Wait()
}
