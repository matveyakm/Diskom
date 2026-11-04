import {FormBlock, FormBlockRow, FormField, RawColumnItem, RawFormItem} from "@/entities/document/types";
import {TableColumnConfig} from "@/shared/DataTable/types";

export const parseTableColumns = (rawColumnsStr: string | undefined) => {
    if (!rawColumnsStr) return [];
    try {
        const rawJson = JSON.parse(rawColumnsStr) as { columns: RawColumnItem[] };

        if (rawJson && Array.isArray(rawJson.columns)) {
            return rawJson.columns
                .filter((c) => c.field !== "status" || c.type === "status")
                .map((c) => ({
                    key: c.field,
                    label: c.label,
                    type: c.type === "status" ? "status" : c.type === "file" ? "file" : "text",
                    end_point: c.end_point,
                })) as TableColumnConfig[];
        }
        return [];
    } catch (e) {
        console.error("Критическая ошибка при JSON.parse(table_columns):", e);
        return [];
    }
};

export const parseFormFields = (rawFieldsStr: string | undefined): FormField[] => {
    if (!rawFieldsStr) return [];
    try {
        const rawJson = JSON.parse(rawFieldsStr) as { fields: FormField[] };

        if (rawJson && Array.isArray(rawJson.fields)) {
            return rawJson.fields;
        }
        return [];
    } catch (e) {
        console.error("Критическая ошибка при JSON.parse(item_fields):", e);
        return [];
    }
};

// Блоки формы (опциональный слой поверх плоских fields).
// Битые блоки отбрасываются по одному, форма при этом живёт.
export const parseFormBlocks = (rawFieldsStr: string | undefined): FormBlock[] => {
    if (!rawFieldsStr) return [];
    let rawJson: RawFormItem;
    try {
        rawJson = JSON.parse(rawFieldsStr) as RawFormItem;
    } catch (e) {
        console.error("Критическая ошибка при JSON.parse(item_fields.blocks):", e);
        return [];
    }
    if (!rawJson || !Array.isArray(rawJson.blocks)) return [];
    const known = new Set((rawJson.fields ?? []).map(f => f.field));
    const seen = new Set<string>();
    const out: FormBlock[] = [];
    for (const b of rawJson.blocks) {
        if (!b || typeof b.title !== "string" || !Array.isArray(b.fields)) {
            console.error("Некорректный блок формы, пропущен:", b);
            continue;
        }
        const inBlock = new Set<string>();
        const fields = b.fields.filter(f => {
            if (typeof f !== "string" || !known.has(f)) {
                console.error(`Блок "${b.title}": неизвестное поле "${String(f)}", пропущено`);
                return false;
            }
            if (seen.has(f)) {
                console.error(`Блок "${b.title}": поле "${f}" уже показано выше, пропущено`);
                return false;
            }
            seen.add(f);
            inBlock.add(f);
            return true;
        });
        if (fields.length === 0) continue;
        // Необязательная табличная раскладка: строки ссылаются на поля блока.
        let rows: FormBlockRow[] | undefined;
        if (b.rows !== undefined) {
            if (!Array.isArray(b.rows)) {
                console.error(`Блок "${b.title}": rows должно быть массивом, игнорирую`);
            } else {
                rows = [];
                for (const r of b.rows) {
                    if (!r || typeof r.label !== "string" || !Array.isArray(r.fields)) {
                        console.error(`Блок "${b.title}": некорректная строка таблицы, пропущена:`, r);
                        continue;
                    }
                    const rowFields = r.fields.filter((f: unknown) => {
                        if (typeof f !== "string" || !inBlock.has(f)) {
                            console.error(`Блок "${b.title}": строка "${r.label}": поле "${String(f)}" вне блока, пропущено`);
                            return false;
                        }
                        return true;
                    });
                    if (rowFields.length === 0) continue;
                    rows.push({label: r.label, fields: rowFields});
                }
                if (rows.length === 0) rows = undefined;
            }
        }
        const columns = Array.isArray(b.columns) && b.columns.every((c: unknown) => typeof c === "string")
            ? (b.columns as string[])
            : undefined;
        out.push({id: b.id, title: b.title, tag: b.tag, fields, columns, rows});
    }
    return out;
};
