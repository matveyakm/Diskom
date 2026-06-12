import React, {useCallback, useState} from "react";
import {useEscapeKey} from "@/shared/hooks/useEscapeKey";
import {TableRow} from "@/shared/DataTable/types";

export type ContextMenuState<T extends TableRow = TableRow> = {
    x: number;
    y: number;
    visible: boolean;
    row: T | null;
};

export function useContextMenu<T extends TableRow = TableRow>() {
    const [menu, setMenu] = useState<ContextMenuState<T>>({
        x: 0,
        y: 0,
        visible: false,
        row: null,
    });

    const openMenu = (e: React.MouseEvent, row: T) => {
        e.preventDefault();

        setMenu({
            x: e.clientX,
            y: e.clientY,
            visible: true,
            row,
        });
    };

    const closeMenu = useCallback(() => {
        setMenu(prev => ({...prev, visible: false}));
    }, []);

    useEscapeKey(closeMenu);

    return {menu, openMenu, closeMenu};
}