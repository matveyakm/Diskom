import {FieldRow} from "@/shared/DataTable/ui/TableModal/FieldRow";
import styles from "@/shared/DataTable/ui/TableModal/TableModal.module.scss";
import {BlockRenderProps} from "./blockRegistry";

// Табличка "падеж × ФИО": строки и колонки задаются в конфиге блока
// (rows: [{label, fields}], columns: ["Ф", "И", "О"]).
// Поля вне строк рисуются обычным списком под таблицей.
export const DeclensionTable = ({block, fields, getValue, isEditing, errors, serviceRef}: BlockRenderProps) => {
    const byName = new Map(fields.map(f => [f.field, f]));
    const covered = new Set((block.rows ?? []).flatMap(r => r.fields));
    const rest = fields.filter(f => !covered.has(f.field));
    return (
        <>
            <table className={styles.blockTable}>
                {block.columns && (
                    <thead>
                        <tr>
                            <th/>
                            {block.columns.map(c => <th key={c}>{c}</th>)}
                        </tr>
                    </thead>
                )}
                <tbody>
                    {(block.rows ?? []).map(row => (
                        <tr key={row.label}>
                            <th>{row.label}</th>
                            {row.fields.map(name => {
                                const field = byName.get(name);
                                if (!field) return null;
                                return (
                                    <td key={name} className={errors[name] ? styles.fieldError : ""}>
                                        <FieldRow field={field} value={getValue(name)} isEditing={isEditing} error={errors[name]} bare serviceRef={serviceRef}/>
                                    </td>
                                );
                            })}
                        </tr>
                    ))}
                </tbody>
            </table>
            {rest.map(field => (
                <FieldRow
                    key={field.field}
                    field={field}
                    value={getValue(field.field)}
                    isEditing={isEditing}
                    error={errors[field.field]}
                    bare
                    serviceRef={serviceRef}
                />
            ))}
        </>
    );
};
