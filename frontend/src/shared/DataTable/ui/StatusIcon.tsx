import SavedIcon from "@/assets/icons/status_saved.svg?react";
import ConductedIcon from "@/assets/icons/status_conducted.svg?react";
import DeletedIcon from "@/assets/icons/status_deleted.svg?react";
import styles from "@/shared/DataTable/DataTable.module.scss";

const STATUS_META: Record<number, { label: string; Icon: typeof SavedIcon; className: string }> = {
    1: {label: "Сохранён", Icon: SavedIcon, className: styles.statusSaved},
    2: {label: "Проведён", Icon: ConductedIcon, className: styles.statusConducted},
    0: {label: "Удалён", Icon: DeletedIcon, className: styles.statusDeleted},
};

// Иконка статуса вместо голой цифры. Неизвестное/пустое значение —
// сырым текстом как раньше (fail-safe, ничего не прячем).
export const StatusIcon = ({value}: { value: unknown }) => {
    if (value === null || value === undefined || value === "") return "";
    const key = typeof value === "number" ? value : Number(value);
    if (!Number.isInteger(key)) return String(value);
    const meta = STATUS_META[key];
    if (!meta) return String(value);
    const {Icon} = meta;
    return (
        <span className={`${styles.statusIcon} ${meta.className}`} title={meta.label}>
            <Icon/>
        </span>
    );
};
