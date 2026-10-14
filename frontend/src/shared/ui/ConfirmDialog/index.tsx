import { createPortal } from "react-dom";
import type { ReactNode } from "react";
import styles from "./ConfirmDialog.module.scss";

interface ConfirmDialogProps {
    open: boolean;
    title: string;
    message: ReactNode;
    confirmText?: string;
    cancelText?: string;
    hideCancel?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

export const ConfirmDialog = ({
    open,
    title,
    message,
    confirmText = "Да",
    cancelText = "Отмена",
    hideCancel = false,
    onConfirm,
    onCancel,
}: ConfirmDialogProps) => {
    if (!open) return null;

    return createPortal(
        <div className={styles.overlay} onClick={hideCancel ? undefined : onCancel}>
            <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
                <h3 className={styles.title}>{title}</h3>
                <div className={styles.message}>{message}</div>
                <div className={styles.actions}>
                    {!hideCancel && (
                        <button className={styles.cancelBtn} onClick={onCancel}>
                            {cancelText}
                        </button>
                    )}
                    <button className={styles.confirmBtn} onClick={onConfirm}>
                        {confirmText}
                    </button>
                </div>
            </div>
        </div>,
        document.body,
    );
};

export type { ConfirmDialogProps };
