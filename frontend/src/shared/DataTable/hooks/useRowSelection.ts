import {useCallback, useState} from "react";
import {TableRow} from "@/shared/DataTable/types";

export function useRowSelection<T extends TableRow = TableRow>() {
    const [activeRow, setActiveRow] = useState<T | null>(null);

    const selectRow = useCallback((row: T) => {
        setActiveRow(row);
    }, []);

    const clearSelection = useCallback(() => {
        setActiveRow(null);
    }, []);

    return {activeRow, selectRow, clearSelection, setActiveRow};
}