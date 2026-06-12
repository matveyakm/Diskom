import styles from "@/shared/DataTable/ui/ContextMenu/ContextMenu.module.scss";
import Tree from "@/assets/icons/account_tree.svg?react";
import Edit from "@/assets/icons/edit.svg?react";
import Print from "@/assets/icons/printer.svg?react";
import type {ContextMenuState} from "@/shared/DataTable/hooks/useContextMenu";
import {useOnClickOutside} from "usehooks-ts";
import React, {useRef} from "react";
import { asElementRef } from "@/shared/utils/refUtils";

type Props = {
    menu: ContextMenuState;
    onClose: () => void;
    onEdit: () => void;
}

const ContextMenu = ({menu, onClose, onEdit}: Props) => {
    const menuRef = useRef<HTMLDivElement>(null);

    useOnClickOutside(
        asElementRef(menuRef),
        () => {
            if (menu.visible) onClose();
        }
    );

    if (!menu.visible) return null;

    return (
        <div
            ref={menuRef}
            className={styles.contextMenu}
            style={{top: menu.y, left: menu.x}}
        >
            <div className={styles.contextBtn}
                 onClick={(e) => {
                     e.stopPropagation();
                     onEdit();
                     onClose();
                     window.getSelection()?.removeAllRanges();
                 }}>
                <Edit/>
            </div>
            <div className={styles.contextBtn}
                onClick={(e) => {
                    e.stopPropagation();
                    onClose();
                }}>
                <Tree/>
            </div>
            <div className={styles.contextBtn}
                 onClick={(e) => {
                     e.stopPropagation();
                     onClose();
                 }}>
                <Print/>
            </div>
        </div>
    );
};

export default ContextMenu;