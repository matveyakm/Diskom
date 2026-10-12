import styles from "./CustomToolbar.module.scss";
import Edit from "@/assets/icons/edit.svg?react";
import Tree from "@/assets/icons/account_tree.svg?react";
import Print from "@/assets/icons/printer.svg?react";
import Next from "@/assets/icons/double_arrow_right.svg?react";
import Add from "@/assets/icons/add.svg?react";
import Filter from "@/assets/icons/filter.svg?react";
import type {TableRow} from "@/shared/DataTable/types";

type Props = {
    onAdd: () => void;
    onEdit: () => void;
    selectedRow: TableRow | null;
    tableName?: string;
    totalCount?: number;
    activeFilterCount?: number;
    onOpenFilters?: () => void;
};

const CustomToolbar = ({onAdd, onEdit, selectedRow, tableName, totalCount, activeFilterCount, onOpenFilters}: Props) => {
    return (
        <div className={styles.toolbar}>
            <span className={styles.tableName}>
                {tableName}
                {totalCount != null && (
                    <span className={styles.count}>Найдено: {totalCount}</span>
                )}
            </span>

            <div className={styles.actionsWrapper}>
                {onOpenFilters && (
                    <button type="button" className={styles.filterButton} onClick={onOpenFilters}>
                        <Filter/>
                        Фильтры
                        {(activeFilterCount ?? 0) > 0 && (
                            <span className={styles.filterBadge}>{activeFilterCount}</span>
                        )}
                    </button>
                )}
                <div className={`${styles.editNav} ${selectedRow ? styles.visible : ""}`}>
                    <div className={styles.modalBtn}
                         onClick={(e) => {
                             e.stopPropagation();
                             onEdit();
                             window.getSelection()?.removeAllRanges();
                         }}>
                        <Edit/>
                    </div>
                    <div className={styles.modalBtn}><Tree/></div>
                    <div className={styles.modalBtn}><Print/></div>
                    <div className={styles.modalBtn}><Next/></div>
                </div>

                <button type="button" className={styles.addButton} onClick={onAdd}>
                    <Add />
                    Добавить
                </button>
            </div>
        </div>
    );
};

export default CustomToolbar;
