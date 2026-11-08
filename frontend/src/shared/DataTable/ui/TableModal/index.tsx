import {useMemo, useState} from "react";
import type {ReactNode} from "react";
import {useQueryClient} from "@tanstack/react-query";
import styles from "./TableModal.module.scss";
import {TableModalProps} from "./types";
import {FormField} from "@/entities/document/types";
import {resolveBlockRenderer} from "./blocks/blockRegistry";
import {getRowValue} from "./utils";
import {useDocumentForm} from "./useDocumentForm";
import {useSaveDocument} from "./useSaveDocument";
import {describePipelineEffects, formatPipelineEffect, parseActionConfig, describeActionError} from "@/shared/api/actionPipeline";
import type {MenuItem} from "@/shared/api/fetchMenuData/types";
import type {PipelineEffect} from "@/shared/api/actionPipeline";
import {useDeleteDocument} from "./useDeleteDocument";
import {useAcceptDocument} from "./useAcceptDocument";
import {FieldRow} from "./FieldRow";
import {ConfirmDialog} from "@/shared/ui/ConfirmDialog";
import {PrintModal} from "@/shared/DataTable/ui/PrintModal";
import {TableRow} from "@/shared/DataTable/types";
import Edit from "@/assets/icons/edit.svg?react";
import Print from "@/assets/icons/printer.svg?react";
import Tree from "@/assets/icons/account_tree.svg?react";
import Delete from "@/assets/icons/delete.svg?react";
import Check from "@/assets/icons/check.svg?react";

type DialogState =
    | { type: "confirm"; title: string; message: ReactNode; onConfirm: () => void }
    | { type: "alert"; title: string; message: ReactNode }
    | null;

// Превью последствий пайплайна: выводится из конфига (поведение — в данных).
// Имена сущностей — из кеша menuData (table_title → name → сырой end_point,
// fail-open, новых запросов нет). Пустой конфиг → пустой список, вызывающий
// показывает плоский текст как раньше.
const usePipelinePreview = (rawConfig: string | null | undefined) => {
    const queryClient = useQueryClient();
    return useMemo(() => {
        const steps = parseActionConfig(rawConfig);
        if (!steps?.length) return [];
        const menuData = queryClient.getQueryData<MenuItem[]>(["menuData"]);
        return describePipelineEffects(steps, (endPoint) => {
            const item = menuData?.find(i => i.ref === endPoint);
            return item?.table_title || item?.name || endPoint;
        });
    }, [queryClient, rawConfig]);
};

const pipelineMessage = (question: string, consequences: string, effects: PipelineEffect[]): ReactNode =>
    effects.length === 0
        ? question
        : (
            <>
                <div>{question} {consequences}</div>
                <ul>
                    {effects.map((e, i) => (
                        <li key={`${e.endPoint}-${i}`}>{formatPipelineEffect(e)}</li>
                    ))}
                </ul>
            </>
        );

const TableModal = <T extends TableRow = TableRow>({selectedRow, onClose, docName, fields, blocks, mode, tableMeta, onSavedRow}: TableModalProps<T>) => {
    const [dialog, setDialog] = useState<DialogState>(null);
    const [printOpen, setPrintOpen] = useState(false);

    const visibleFields = fields.filter(f => !f.hidden);
    const itemName = tableMeta?.item_title;

    // Группировка для отображения. Вся логика (валидация, сбор payload,
    // дефолты, пайплайны) по-прежнему работает с плоским visibleFields выше —
    // блоки меняют только раскладку.
    const visibleByName = useMemo(() => new Map(visibleFields.map(f => [f.field, f])), [visibleFields]);
    const visibleBlocks = useMemo(
        () => (blocks ?? [])
            .map(b => {
                const fields = b.fields.filter(name => visibleByName.has(name));
                const rows = b.rows
                    ?.map(r => ({...r, fields: r.fields.filter(name => visibleByName.has(name))}))
                    .filter(r => r.fields.length > 0);
                return {...b, fields, rows: rows && rows.length > 0 ? rows : undefined};
            })
            .filter(b => {
                if (b.fields.length === 0) return false;
                if (b.tag && !resolveBlockRenderer(b.tag)) {
                    console.warn(`[form] блок "${b.title}": неизвестный тег "${b.tag}", обычный рендер`);
                }
                return true;
            }),
        [blocks, visibleByName],
    );
    const groupedNames = useMemo(() => new Set(visibleBlocks.flatMap(b => b.fields)), [visibleBlocks]);
    const ungroupedFields = useMemo(
        () => visibleFields.filter(f => !groupedNames.has(f.field)),
        [visibleFields, groupedNames],
    );

    const acceptanceEffects = usePipelinePreview(tableMeta?.on_acceptance);
    const deleteEffects = usePipelinePreview(tableMeta?.on_delete);
    const acceptanceMessage = pipelineMessage(
        "Провести документ?", "Последствия проведения:", acceptanceEffects,
    );
    const deleteMessage = pipelineMessage(
        "Вы уверены, что хотите удалить эту запись?", "Последствия удаления:", deleteEffects,
    );

    const renderField = (field: FormField) => (
        <div
            key={field.field}
            className={`${styles.fieldWrapper} ${styles.viewMode} ${errors[field.field] ? styles.fieldError : ""}`}
        >
            <FieldRow
                field={field}
                value={getRowValue(selectedRow, field.field)}
                isEditing={isEditing}
                error={errors[field.field]}
                serviceRef={docName}
                docId={selectedRow?.id}
            />
        </div>
    );

    const showConfirm = (message: ReactNode, onConfirm: () => void) => {
        setDialog({type: "confirm", title: "Подтверждение", message, onConfirm});
    };

    const showAlert = (message: ReactNode) => {
        setDialog({type: "alert", title: "Ошибка", message});
    };

    const closeDialog = () => setDialog(null);

    const {mutate: saveRow, mutateAsync: saveRowAsync, isPending} = useSaveDocument({
        docName, mode, visibleFields, tableMeta,
    });

    const {mutate: deleteRow, isPending: isDeleting} = useDeleteDocument<T>({
        docName, selectedRow, tableMeta,
        onSuccess: onClose,
        onError: () => showAlert("Не удалось удалить запись."),
    });

    const {mutate: acceptRow, mutateAsync: acceptRowAsync, isPending: isAccepting} = useAcceptDocument<T>({
        docName, selectedRow, fields, tableMeta,
    });

    const {errors, setErrors, isEditing, setIsEditing, handleSubmit} = useDocumentForm<T>({
        visibleFields, mode, selectedRow,
        onSave: (payload, action) => {
            if (action === "saveAccept") {
                showConfirm(acceptanceMessage, () => {
                    void handleSaveAndAccept(payload);
                });
            } else {
                handleSave(payload);
            }
        },
    });

    // После сохранения — просмотр записанной строки вместо закрытия модалки:
    // серверное эхо (id, dnumber, дефолты) честнее формы. Родитель переключает
    // модалку на эту строку; без обработчика — старое поведение (закрыть).
    const showSavedRow = (saved: Record<string, unknown>) => {
        setErrors({});
        setIsEditing(false);
        if (onSavedRow && saved != null && typeof saved === "object") {
            onSavedRow(saved as unknown as T);
        } else {
            onClose();
        }
    };

    const handleSave = (payload: Record<string, unknown>) => {
        saveRow(payload, {
            onSuccess: (saved) => showSavedRow(saved ?? {}),
            onError: () => showAlert("Не удалось сохранить изменения. Проверьте введенные данные."),
        });
    };

    const handleSaveAndAccept = async (payload: Record<string, unknown>) => {
        let saved: Record<string, unknown>;
        try {
            saved = await saveRowAsync(payload);
        } catch {
            showAlert("Не удалось сохранить изменения. Проверьте введенные данные.");
            return;
        }
        const savedId = saved.id ?? (selectedRow as TableRow | null)?.id;
        if (savedId == null) {
            showAlert("Документ сохранён, но провести не удалось: нет ID записи.");
            return;
        }
        const rowForAccept = {
            ...((selectedRow ?? {}) as Record<string, unknown>),
            ...saved,
            id: savedId,
        } as unknown as T;
        try {
            await acceptRowAsync(rowForAccept);
        } catch (err) {
            showAlert(`Документ сохранён, но провести не удалось (${describeActionError(err)}). Проверьте данные и проведите позже.`);
            return;
        }
        // Проведение ставит status 2 (терминальный шаг всех on_acceptance) —
        // показываем проведённый документ, а не табличку.
        showSavedRow({...rowForAccept, status: 2} as unknown as Record<string, unknown>);
    };

    const handleCancel = () => {
        setErrors({});
        if (mode === "create") {
            onClose();
        } else {
            setIsEditing(false);
        }
    };

    const handleDelete = () => {
        showConfirm(deleteMessage, () => {
            deleteRow();
        });
    };

    const handleAccept = () => {
        showConfirm(acceptanceMessage, () => {
            acceptRow(undefined, {
                onSuccess: () => {
                    if (selectedRow && onSavedRow) {
                        showSavedRow({...selectedRow, status: 2} as unknown as Record<string, unknown>);
                    } else {
                        onClose();
                    }
                },
                onError: (err) => showAlert(`Не удалось провести документ (${describeActionError(err)}).`),
            });
        });
    };

    const isBusy = isPending || isAccepting;

    // Кнопка "Провести" — только если проведение сконфигурировано:
    // иначе клик уходил во fallback и молча ставил status=2 где попало.
    const hasAcceptancePipeline = (parseActionConfig(tableMeta?.on_acceptance)?.length ?? 0) > 0;
    const canAccept =
        Number((selectedRow as unknown as Record<string, unknown>)?.status) === 1 &&
        hasAcceptancePipeline;
    // "Сохранить и провести": в создании — по наличию конфига (статуса ещё нет),
    // в правке — по тем же правилам, что у "Провести". Где проводить нечего
    // (справочники, dismissal) — кнопки нет, чтобы не портить данные.
    const showSaveAccept = mode === "create" ? hasAcceptancePipeline : canAccept;

    // Печать — там же, где проведение (документы/отчёты): у справочников
    // и регистров проводить нечего и печатать нечего. Отдельного флага нет —
    // capability выводится из того же конфига, без веток по типу сущности.
    // (У сущностей без шаблона печати кнопка тоже прячется — вместе с пайплайном.)
    const canPrint = hasAcceptancePipeline;

    return (
        <>
            <div className={styles.overlay} onClick={onClose}/>
            <div className={styles.modal}>
                <header className={styles.header}>
                    <div className={styles.titleInfo}>
                        <h3 className={styles.heading}>
                            {mode === "create" ? `Создание: ${itemName}` : itemName}
                        </h3>
                    </div>
                    <button className={styles.closeX} onClick={onClose} aria-label="Закрыть">
                        ✕
                    </button>
                </header>

                {!isEditing && mode === "edit" && (
                    <div className={styles.toolbar}>
                        {canAccept && (
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
                        {canPrint && (
                            <button type="button" className={styles.actionBtn} onClick={() => setPrintOpen(true)}>
                                <div className={styles.icon}><Print/></div> Печать
                            </button>
                        )}
                        <button type="button" className={styles.actionBtn}>
                            <div className={styles.icon}><Tree/></div> Дерево
                        </button>
                    </div>
                )}

                <div className={styles.content}>
                    <form id="modal-form" className={styles.formContainer} onSubmit={handleSubmit}>
                        <div className={styles.fieldsStack}>
                            {ungroupedFields.map(renderField)}
                            {visibleBlocks.map((block, blockIdx) => {
                                const Renderer = resolveBlockRenderer(block.tag);
                                const blockKey = block.id ?? `${block.title}-${blockIdx}`;
                                if (!Renderer) {
                                    return (
                                        <section key={blockKey} className={styles.blockSection}>
                                            <h4 className={styles.blockTitle}>{block.title}</h4>
                                            {block.fields.map(name => renderField(visibleByName.get(name)!))}
                                        </section>
                                    );
                                }
                                return (
                                    <section key={blockKey} className={styles.blockSection}>
                                        <h4 className={styles.blockTitle}>{block.title}</h4>
                        <Renderer
                            block={block}
                            fields={block.fields.map(name => visibleByName.get(name)!)}
                            getValue={name => getRowValue(selectedRow, name)}
                            isEditing={isEditing}
                            errors={errors}
                            serviceRef={docName}
                            docId={selectedRow?.id}
                        />
                                    </section>
                                );
                            })}
                        </div>

                        {isEditing && (
                            <div className={styles.formFooter}>
                                <button type="button" onClick={handleCancel} className={styles.cancelBtn}
                                        disabled={isBusy}>
                                    Отмена
                                </button>
                                <button type="submit" className={styles.saveBtn} disabled={isBusy}>
                                    {isPending ? "Сохранение..." : "Сохранить"}
                                </button>
                                {showSaveAccept && (
                                    <button type="submit" data-action="saveAccept" className={styles.saveBtn}
                                            disabled={isBusy}>
                                        {isAccepting ? "Проведение..." : isPending ? "Сохранение..." : "Сохранить и провести"}
                                    </button>
                                )}
                            </div>
                        )}
                    </form>
                </div>
            </div>

            {printOpen && selectedRow != null && (
                <PrintModal
                    docName={docName}
                    rowId={selectedRow.id}
                    docTitle={`${itemName ?? docName} ${getRowValue(selectedRow, "dnumber") ?? ""}`.trim()}
                    onClose={() => setPrintOpen(false)}
                />
            )}

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
