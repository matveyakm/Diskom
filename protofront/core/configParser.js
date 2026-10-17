/* Config parsing (single responsibility: blob -> structured config,
 * no DOM, no network). Pure functions. */
function parseActionConfig(raw) {
  if (!raw || raw === "") return null;
  try {
    const config = typeof raw === "string" ? JSON.parse(raw) : raw;
    return Array.isArray(config.processing) ? config.processing : null;
  } catch (e) {
    console.error("ConfigParser: ошибка парсинга конфига:", e);
    return null;
  }
}

function parseBlob(raw) {
  // Already parsed (new shape) or absent ('' means "no config").
  if (!raw) return null;
  if (typeof raw !== "string") return raw;
  if (raw.trim() === "") return null;
  return JSON.parse(raw);
}

function parseColumnsConfig(jsonColumns, jsonItemFields) {
  return {
    columns: parseBlob(jsonColumns)?.columns ?? [],
    itemFields: parseBlob(jsonItemFields)?.fields ?? [],
    itemBlocks: parseBlob(jsonItemFields)?.blocks ?? [],
  };
}
