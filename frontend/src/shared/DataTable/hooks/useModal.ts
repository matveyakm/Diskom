import { useCallback, useState } from "react";
import { useEscapeKey } from "@/shared/hooks/useEscapeKey";
import type {TableRow} from "@/shared/DataTable/types";

export function useModal<T extends TableRow = TableRow>() {
    const [isOpen, setIsOpen] = useState(false);
    const [data, setData] = useState<T | null>(null);
    const [mode, setMode] = useState<"edit" | "create">("edit");

    const openForEdit = (value: T) => {
        setMode("edit");
        setData(value);
        setIsOpen(true);
    };

    const openForCreate = () => {
        setMode("create");
        setData({} as T);
        setIsOpen(true);
    };

    const close = useCallback(() => {
        setIsOpen(false);
        setData(null);
    }, []);

    useEscapeKey(close, isOpen);

    return { isOpen, data, mode, openForEdit, openForCreate, close };
}