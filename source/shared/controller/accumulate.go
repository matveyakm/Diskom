package restController

// Накапливаемая сумма — общий shared-механизм для регистров-остатков.
//
// Канал включения — env `ACCUMULATE=balance:delta[:order]` (как DNUMBER_PREFIX:
// данные, а не код; ни одного имени сервиса/колонки здесь нет). Сегодня
// использует один сервис (employee_data_ledger), завтра — любой другой
// одной строкой env.
//
// Контракт: запрос присылает только дельту (`delta`: +1/-1), сервер в той же
// транзакции пишет накопленный итог в `balance` и всегда его пересчитывает
// целиком (не «последний + n»). Полный пересчёт закрывает вставку задним
// числом и удаление из середины истории — обе операции у нас есть
// (проведение задним числом, on_delete чистит проводки по id_document).
// Порядок накопления — `(order, id)`; `order` по умолчанию `id`.
// Явный `balance` в теле — 400 (единственный вход — дельта).
// Прямая запись без дельты — 400 (регистры руками не пишутся).
// Кривой spec — громкий 500 на первом запросе (fail-closed на своём синтаксисе).

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// accumulateEnvName — единственный конфигурационный knob механизма.
const accumulateEnvName = "ACCUMULATE"

// accumulateSpec — разобранный spec: куда копим, откуда берём дельту,
// по какой колонке упорядочиваем историю (пусто = id).
type accumulateSpec struct {
	balance string
	delta   string
	order   string
}

// parseAccumulateSpec разбирает `balance:delta[:order]`. Чистая, с тестами.
func parseAccumulateSpec(spec string) (accumulateSpec, bool) {
	parts := strings.Split(strings.TrimSpace(spec), ":")
	if len(parts) != 2 && len(parts) != 3 {
		return accumulateSpec{}, false
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if !reFilterColumn.MatchString(parts[i]) {
			return accumulateSpec{}, false
		}
	}
	s := accumulateSpec{balance: parts[0], delta: parts[1], order: "id"}
	if len(parts) == 3 {
		s.order = parts[2]
	}
	if s.balance == s.delta {
		return accumulateSpec{}, false
	}
	return s, true
}

// extractAccumulateDelta достаёт целую дельту из map-строки (round-trip
// JSON-декода: числа — float64). Чистая, с тестами. Нецелое/bool/null —
// ошибка клиента.
func extractAccumulateDelta(row map[string]any, deltaCol string) (int64, error) {
	raw, has := row[deltaCol]
	if !has || raw == nil {
		return 0, fmt.Errorf("missing %q", deltaCol)
	}
	switch t := raw.(type) {
	case float64:
		if n := int64(t); float64(n) == t {
			return n, nil
		}
	case int:
		return int64(t), nil
	case int64:
		return t, nil
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
			return n, nil
		}
	}
	return 0, fmt.Errorf("invalid %q: want integer delta", deltaCol)
}

// lockAccumulateScope сериализует параллельные записи скоупа: второй писатель
// ждёт конца транзакции первого и пересчитывает поверх. Ключ именован
// таблицей (данные), коллизий с чужими lock нет по построению.
func lockAccumulateScope(tx *gorm.DB, table string) error {
	var locked int
	// table — из env оператора (DB_TABLENAME), не из запроса; параметром,
	// не интерполяцией.
	if err := tx.Raw("SELECT pg_advisory_xact_lock(hashtext(?))", "accumulate:"+table).Scan(&locked).Error; err != nil {
		return fmt.Errorf("accumulate lock: %w", err)
	}
	return nil
}

// recomputeAccumulate пересчитывает balance всего скоупа одним UPDATE
// с оконной суммой дельт. Колонки уже провалидированы регекспом в spec
// (тот же приём, что фильтры/updateColumns), интерполяция безопасна.
func recomputeAccumulate(tx *gorm.DB, table string, spec accumulateSpec) error {
	// Порядок: NULLS LAST, чтобы строки без даты не ломали историю.
	q := fmt.Sprintf(
		"UPDATE %s SET %s = s.b FROM (SELECT id, SUM(%s) OVER (ORDER BY %s NULLS LAST, id) AS b FROM %s) s WHERE %s.id = s.id",
		table, spec.balance, spec.delta, spec.order, table, table,
	)
	if err := tx.Exec(q).Error; err != nil {
		return fmt.Errorf("accumulate recompute: %w", err)
	}
	return nil
}

// readAccumulateBalance читает свежепосчитанный итог строки (для эха в ответе).
func readAccumulateBalance(tx *gorm.DB, table string, spec accumulateSpec, id int64) (int64, error) {
	var balance int64
	if err := tx.Table(table).Where("id = ?", id).Pluck(spec.balance, &balance).Error; err != nil {
		return 0, fmt.Errorf("accumulate reread: %w", err)
	}
	return balance, nil
}

// accumulateActive различает «выключено» (env нет) и «сломано» (env есть,
// spec кривой — громкий 500, fail-closed на своём синтаксисе).
func accumulateActive() (accumulateSpec, bool, error) {
	raw, set := os.LookupEnv(accumulateEnvName)
	if !set {
		return accumulateSpec{}, false, nil
	}
	spec, ok := parseAccumulateSpec(raw)
	if !ok {
		return accumulateSpec{}, true, fmt.Errorf("invalid %s %q: want balance:delta[:order]", accumulateEnvName, raw)
	}
	return spec, true, nil
}

// accumulateModelOK проверяет, что обе колонки есть в схеме модели сервиса
// (тот же приём, что updateColumns). Без этого recompute упал бы SQL-ошибкой
// уже внутри транзакции — лучше громко и заранее, до записи.
func (r *CRUDRepository[T]) accumulateModelOK(spec accumulateSpec) bool {
	var zero T
	stmt := &gorm.Statement{DB: r.db}
	if err := stmt.Parse(&zero); err != nil {
		return false
	}
	has := func(col string) bool {
		if _, ok := stmt.Schema.FieldsByDBName[col]; ok {
			return true
		}
		_, ok := stmt.Schema.FieldsByName[col]
		return ok
	}
	return has(spec.balance) && has(spec.delta)
}

// reflectAccumulateBalance подшивает посчитанный итог обратно в entity
// для эха в ответе (тот же map round-trip, что fillDnumber).
func reflectAccumulateBalance(entity any, balanceCol string, balance int64) {
	raw, err := json.Marshal(entity)
	if err != nil {
		return
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return
	}
	row[balanceCol] = balance
	if back, err := json.Marshal(row); err == nil {
		_ = json.Unmarshal(back, entity)
	}
}

// accumulateEntityID достаёт положительный id сущности (автоинкремент
// GORM проставляет в структуру при Create; явный id — заранее).
func accumulateEntityID(entity any) (int64, bool) {
	type identifiable interface{ GetID() int64 }
	if id, ok := entity.(identifiable); ok {
		if n := id.GetID(); n > 0 {
			return n, true
		}
	}
	return 0, false
}

// rejectAccumulateBalance — 400, если тело явно прислало balance.
// Единственный вход при включённом механизме — дельта.
func rejectAccumulateBalance(sent map[string]any, spec accumulateSpec) error {
	if _, has := sent[spec.balance]; has {
		return fmt.Errorf("%q is server-computed, send %q instead", spec.balance, spec.delta)
	}
	return nil
}

// createAccumulated — POST при заданном ACCUMULATE: тело нужно и картой
// (presence-check balance, честная 400), и структурой. Без spec вызывается
// старый путь Create без изменений — горячий путь остальных сервисов не тронут.
func (r *CRUDRepository[T]) createAccumulated(w http.ResponseWriter, req *http.Request, spec accumulateSpec) {
	rawBody, err := io.ReadAll(http.MaxBytesReader(w, req.Body, updateMaxBodyBytes))
	if err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	var sent map[string]any
	if err := json.Unmarshal(rawBody, &sent); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var entity T
	if err := json.Unmarshal(rawBody, &entity); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Fail-closed входы: баланс руками запрещён, дельта обязана быть целой.
	if err := rejectAccumulateBalance(sent, spec); err != nil {
		r.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := extractAccumulateDelta(sent, spec.delta); err != nil {
		r.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !r.accumulateModelOK(spec) {
		r.writeError(w, "Accumulate columns not in model", http.StatusInternalServerError)
		return
	}

	tx := r.db.Begin()
	if tx.Error != nil {
		r.writeError(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	if err := lockAccumulateScope(tx, r.tableName); err != nil {
		tx.Rollback()
		r.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Table(r.tableName).Create(&entity).Error; err != nil {
		tx.Rollback()
		// Сюда же падает повторное проведение (UNIQUE) — громко, как задумано.
		r.writeError(w, "Failed to create record: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Автогенерация "Кода" — как в обычном Create (там no-op без префикса).
	if prefix := dnumberPrefix(); prefix != "" {
		if err := fillDnumber(tx, r.tableName, &entity, prefix); err != nil {
			tx.Rollback()
			r.writeError(w, "Failed to assign document number: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := recomputeAccumulate(tx, r.tableName, spec); err != nil {
		tx.Rollback()
		r.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Эхо: подшиваем посчитанный итог в ответ (GORM уже проставил id).
	if id, ok := accumulateEntityID(any(&entity)); ok {
		if balance, err := readAccumulateBalance(tx, r.tableName, spec, id); err != nil {
			tx.Rollback()
			r.writeError(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			reflectAccumulateBalance(&entity, spec.balance, balance)
		}
	}
	if err := tx.Commit().Error; err != nil {
		r.writeError(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	r.writeJSONResponse(w, entity, http.StatusCreated)
}

// refreshAccumulated — lock скоупа + пересчёт после успешной записи/правки/
// удаления (порядок истории мог сдвинуться через ddate, дельту могли
// поправить, строки — удалиться). Вызывать внутри tx вызывателя, до коммита.
func (r *CRUDRepository[T]) refreshAccumulated(tx *gorm.DB, spec accumulateSpec) error {
	if err := lockAccumulateScope(tx, r.tableName); err != nil {
		return err
	}
	return recomputeAccumulate(tx, r.tableName, spec)
}
