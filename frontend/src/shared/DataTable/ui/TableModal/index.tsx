import {useState} from "react";
import styles from "./TableModal.module.scss";
import {TableModalProps} from "./types";
import {getRowValue} from "./utils";
import {useDocumentForm} from "./useDocumentForm";
import {useSaveDocument} from "./useSaveDocument";
import {useDeleteDocument} from "./useDeleteDocument";
import {useAcceptDocument} from "./useAcceptDocument";
import {FieldRow} from "./FieldRow";
import {ConfirmDialog} from "@/shared/ui/ConfirmDialog";
import {TableRow} from "@/shared/DataTable/types";
import Edit from "@/assets/icons/edit.svg?react";
import Print from "@/assets/icons/printer.svg?react";
import Tree from "@/assets/icons/account_tree.svg?react";
import Delete from "@/assets/icons/delete.svg?react";
import Check from "@/assets/icons/check.svg?react";

type DialogState =
    | { type: "confirm"; title: string; message: string; onConfirm: () => void }
    | { type: "alert"; title: string; message: string }
    | null;

const TableModal = <T extends TableRow = TableRow>({selectedRow, onClose, docName, fields, mode, tableMeta}: TableModalProps<T>) => {
    const [dialog, setDialog] = useState<DialogState>(null);

    const visibleFields = fields.filter(f => !f.hidden);
    const itemName = tableMeta?.item_title;

    const showConfirm = (message: string, onConfirm: () => void) => {
        setDialog({type: "confirm", title: "Подтверждение", message, onConfirm});
    };

    const showAlert = (message: string) => {
        setDialog({type: "alert", title: "Ошибка", message});
    };

    const closeDialog = () => setDialog(null);

    const {mutate: saveRow, isPending} = useSaveDocument({
        docName, mode, visibleFields, tableMeta,
        onSuccess: () => {
            setErrors({});
            setIsEditing(false);
            onClose();
        },
        onError: () => showAlert("Не удалось сохранить изменения. Проверьте введенные данные."),
    });

    const {mutate: deleteRow, isPending: isDeleting} = useDeleteDocument<T>({
        docName, selectedRow, tableMeta,
        onSuccess: onClose,
        onError: () => showAlert("Не удалось удалить запись."),
    });

    const {mutate: acceptRow, isPending: isAccepting} = useAcceptDocument<T>({
        docName, selectedRow, fields, tableMeta,
        onSuccess: onClose,
        onError: () => showAlert("Не удалось провести документ."),
    });

    const {errors, setErrors, isEditing, setIsEditing, handleSubmit} = useDocumentForm<T>({
        visibleFields, mode, selectedRow,
        onSave: saveRow,
    });

    const handleCancel = () => {
        setErrors({});
        if (mode === "create") {
            onClose();
        } else {
            setIsEditing(false);
        }
    };

    const handleDelete = () => {
        showConfirm("Вы уверены, что хотите удалить эту запись?", () => {
            deleteRow();
        });
    };

    const handleAccept = () => {
        showConfirm("Провести документ?", () => {
            acceptRow();
        });
    };

    return (
        <>
            <div className={styles.overlay} onClick={onClose}/>
            <div className={styles.modal}>
                <header className={styles.header}>
                    <div className={styles.titleInfo}>
                        <h3 className={styles.heading}>
                            {mode === "create" ? `Создание: ${itemName}` : itemName}
                        </h3>
                        {mode !== "create" && (
                            <span className={styles.docId}>
                                #{String(selectedRow?.id ?? "")}
                            </span>
                        )}
                    </div>
                    <button className={styles.closeX} onClick={onClose} aria-label="Закрыть">
                        ✕
                    </button>
                </header>

                {!isEditing && mode === "edit" && (
                    <div className={styles.toolbar}>
                        {Number((selectedRow as unknown as Record<string, unknown>)?.status) === 1 && (
                            <button type="button" onClick={handleAccept} className={styles.acceptBtn}
                                    disabled={isAccepting}>
                                <div className={styles.icon}><Check/></div>
                                {isAccepting ? "Проведение..." : "Провести"}
                            </button>
                        )}
                        <button type="button" onClick={() => setIsEditing(true)} className={styles.actionBtn}>
                            <div className={styles.icon}><Edit/></div> Редактировать
                        </button>
                        <button type="button" onClick={handleDelete} className={styles.deleteBtn} disabled={isDeleting}>
                            <div className={styles.icon}><Delete/></div> {isDeleting ? "Удаление..." : "Удалить"}
                        </button>
                        <button type="button" className={styles.actionBtn}>
                            <div className={styles.icon}><Print/></div> Печать
                        </button>
                        <button type="button" className={styles.actionBtn}>
                            <div className={styles.icon}><Tree/></div> Дерево
                        </button>
                    </div>
                )}

                <div className={styles.content}>
                    <form id="modal-form" className={styles.formContainer} onSubmit={handleSubmit}>
                        <div className={styles.fieldsStack}>
                            {visibleFields.map(field => (
                                <div
                                    key={field.field}
                                    className={`${styles.fieldWrapper} ${styles.viewMode} ${errors[field.field] ? styles.fieldError : ""}`}
                                >
                                    <FieldRow
                                        field={field}
                                        value={getRowValue(selectedRow, field.field)}
                                        isEditing={isEditing}
                                        error={errors[field.field]}
                                    />
                                </div>
                            ))}
                        </div>

                        {isEditing && (
                            <div className={styles.formFooter}>
                                <button type="button" onClick={handleCancel} className={styles.cancelBtn}
                                        disabled={isPending}>
                                    Отмена
                                </button>
                                <button type="submit" className={styles.saveBtn} disabled={isPending}>
                                    {isPending ? "Сохранение..." : "Сохранить изменения"}
                                </button>
                            </div>
                        )}
                    </form>
                </div>
            </div>

            {dialog && (
                <ConfirmDialog
                    open
                    title={dialog.title}
                    message={dialog.message}
                    confirmText={dialog.type === "alert" ? "ОК" : "Да"}
                    hideCancel={dialog.type === "alert"}
                    onConfirm={() => {
                        if (dialog.type === "confirm") dialog.onConfirm();
                        closeDialog();
                    }}
                    onCancel={closeDialog}
                />
            )}
        </>
    );
};

export default TableModal;
