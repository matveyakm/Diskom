package restController

// Automatic numbering ("Код") on record creation.
//
// If the service is configured with DNUMBER_PREFIX env (e.g. "НАД-") and the
// created entity carries an empty "dnumber", the number is generated as
// PREFIX + id (e.g. "НАД-42") inside the same transaction, right after insert
// (the id is only known then for auto-increment tables; for tables with an
// explicit id it is known beforehand, but the uniform post-insert path keeps
// a single code path for every model).
//
// Skipped silently when: the entity has no dnumber column (most ledgers),
// the number was provided manually (never overwritten), the env is unset,
// or the id is missing/non-positive. Any failure to persist the generated
// number aborts the whole transaction.

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// dnumberEnvName is the only configuration knob of this feature.
const dnumberEnvName = "DNUMBER_PREFIX"

// dnumberPrefix returns the operator-configured prefix, "" when disabled.
func dnumberPrefix() string {
	return strings.TrimSpace(os.Getenv(dnumberEnvName))
}

// buildDnumber assembles the human-readable number: prefix + decimal id.
func buildDnumber(prefix string, id int64) string {
	return prefix + strconv.FormatInt(id, 10)
}

// dnumberID extracts a positive integer id from a JSON-decoded value.
func dnumberID(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		if id := int64(t); float64(id) == t && id > 0 {
			return id, true
		}
		return 0, false
	case string:
		t = strings.TrimSpace(t)
		if !reDigits.MatchString(t) {
			return 0, false
		}
		id, err := strconv.ParseInt(t, 10, 64)
		if err != nil || id <= 0 {
			return 0, false
		}
		return id, true
	default:
		return 0, false
	}
}

// planDnumber decides whether a number must be generated for the row.
// Pure function, unit-tested.
func planDnumber(row map[string]any, prefix string) (id int64, value string, ok bool) {
	if prefix == "" {
		return 0, "", false
	}
	raw, has := row["dnumber"]
	if !has {
		return 0, "", false
	}
	if s, isStr := raw.(string); isStr && strings.TrimSpace(s) != "" {
		return 0, "", false // manually provided — keep
	} else if raw != nil && !isStr {
		return 0, "", false // unexpected type — do not touch
	}
	id, ok = dnumberID(row["id"])
	if !ok {
		return 0, "", false
	}
	return id, buildDnumber(prefix, id), true
}

// fillDnumber persists the generated number and reflects it into entity
// (so the create response already contains it). Runs inside the caller's tx.
func fillDnumber(tx *gorm.DB, tableName string, entity any, prefix string) error {
	raw, err := json.Marshal(entity)
	if err != nil {
		return fmt.Errorf("encode entity: %w", err)
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return fmt.Errorf("decode entity: %w", err)
	}
	id, value, ok := planDnumber(row, prefix)
	if !ok {
		return nil
	}
	if err := tx.Table(tableName).Where("id = ?", id).Update("dnumber", value).Error; err != nil {
		return fmt.Errorf("store dnumber: %w", err)
	}
	row["dnumber"] = value
	if back, err := json.Marshal(row); err == nil {
		_ = json.Unmarshal(back, entity)
	}
	return nil
}
