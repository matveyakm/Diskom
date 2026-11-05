import {useReferenceOptions, ReferenceOption} from "@/shared/api/useReferenceOptions";
import {api} from "@/shared/api/AxiosClient";
import {FILE_DETACHED_SUFFIX} from "@/entities/document/types";
import {FieldRowProps} from "./types";
import styles from "./TableModal.module.scss";
import {useEffect, useState, useRef, useCallback, useMemo, type ReactNode} from "react";
import {isDateField, formatDateForDisplay, toDateInputValue, getTodayDate} from "./utils";

export const FieldRow = ({field, value, isEditing, error, bare, serviceRef, docId}: FieldRowProps) => {
    const [currentValue, setCurrentValue] = useState(String(value ?? ""));

    useEffect(() => {
        setCurrentValue(String(value ?? ""));
    }, [value]);

    if (field.readonly) {
        return <ReadonlyField field={field} value={value} isEditing={isEditing} error={error} bare={bare}/>;
    }

    if (field.type === "select" && field.end_point) {
        return <SelectField field={field} value={value} isEditing={isEditing} error={error} bare={bare}/>;
    }

    if (isDateField(field)) {
        return <DateField field={field} value={value} isEditing={isEditing} error={error} bare={bare}/>;
    }

    if (field.type === "file") {
        return <FileField field={field} value={value} isEditing={isEditing} error={error} bare={bare} serviceRef={serviceRef} docId={docId}/>;
    }

    const isRequired = field.required ?? false;

    return (
        <>
            {!bare && (
                <label>
                    {field.label} {isRequired && <span style={{color: "red"}}>*</span>}
                </label>
            )}
            {isEditing ? (
                field.type === "textarea" ? (
                    <textarea
                        name={field.field}
                        rows={3}
                        value={currentValue}
                        required={isRequired}
                        onChange={(e) => setCurrentValue(e.target.value)}
                    />
                ) : (
                    <input
                        name={field.field}
                        type="text"
                        value={currentValue}
                        required={isRequired}
                        onChange={(e) => setCurrentValue(e.target.value)}
                    />
                )
            ) : (
                <div className={styles.viewValue}>{String(value ?? "—")}</div>
            )}
        </>
    );
};

const DateField = ({field, value, isEditing, bare}: FieldRowProps) => {
    const [currentValue, setCurrentValue] = useState(() => {
        if (value) return toDateInputValue(value);
        if (isEditing) return getTodayDate();
        return "";
    });

    const isRequired = field.required ?? false;

    return (
        <>
            {!bare && (
                <label>
                    {field.label} {isRequired && <span style={{color: "red"}}>*</span>}
                </label>
            )}
            {isEditing ? (
                <input
                    name={field.field}
                    type="date"
                    value={currentValue}
                    required={isRequired}
                    onChange={(e) => setCurrentValue(e.target.value)}
                />
            ) : (
                <div className={styles.viewValue}>{formatDateForDisplay(value)}</div>
            )}
        </>
    );
};

// ReadonlyField shows the value as text (or a placeholder when empty) and
// always submits it via a hidden input, so the backend stays the source
// of truth (e.g. auto-generated document numbers).
const ReadonlyField = ({field, value, isEditing, bare}: FieldRowProps) => {
    const text = value === null || value === undefined || value === "" ? "" : String(value);
    const emptyText = isEditing ? "Сгенерируется автоматически" : "—";
    return (
        <>
            {!bare && <label>{field.label}</label>}
            <div className={styles.viewValue}>{text !== "" ? text : emptyText}</div>
            <input type="hidden" name={field.field} value={text}/>
        </>
    );
};

const SelectField = ({field, value, isEditing, bare}: FieldRowProps) => {
    const [search, setSearch] = useState("");
    const [open, setOpen] = useState(false);
    const [selectedId, setSelectedId] = useState<string | number>(() =>
        String(value ?? "")
    );
    const wrapperRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    const isRequired = field.required ?? false;

    const {data: allOptions, isLoading, error: queryError} = useReferenceOptions(field.end_point);

    const typedOptions = useMemo<ReferenceOption[]>(() => allOptions ?? [], [allOptions]);

    const filtered = !search.trim()
        ? typedOptions
        : typedOptions.filter(o =>
            o.name.toLowerCase().includes(search.toLowerCase())
        );

    useEffect(() => {
        setSelectedId(String(value ?? ""));
    }, [value]);

    useEffect(() => {
        if (open) return;
        const match = typedOptions.find(o => String(o.id) === String(selectedId));
        if (match) setSearch(match.name);
    }, [typedOptions, selectedId, open]);

    const handleSelect = useCallback((opt: ReferenceOption) => {
        setSelectedId(opt.id);
        setSearch(opt.name);
        setOpen(false);
    }, []);

    useEffect(() => {
        if (!open) return;
        const handler = (e: MouseEvent) => {
            if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
                setOpen(false);
                const match = typedOptions.find(o => String(o.id) === String(selectedId));
                setSearch(match ? match.name : "");
            }
        };
        document.addEventListener("mousedown", handler);
        return () => document.removeEventListener("mousedown", handler);
    }, [open, typedOptions, selectedId]);

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === "Escape") {
            setOpen(false);
            const match = typedOptions.find(o => String(o.id) === String(selectedId));
            setSearch(match ? match.name : "");
            inputRef.current?.blur();
        }
    };

    if (queryError) {
        return (
            <>
                {!bare && <label>{field.label}</label>}
                <div style={{color: "red", fontSize: "12px"}}>Ошибка загрузки справочника</div>
            </>
        );
    }

    if (!isEditing) {
        const selectedOption = typedOptions.find(o => String(o.id) === String(value));
        const displayValue = selectedOption ? selectedOption.name : String(value ?? "—");
        return (
            <>
                {!bare && <label>{field.label}</label>}
                <div className={styles.viewValue}>{displayValue}</div>
            </>
        );
    }

    return (
        <div className={styles.autocompleteWrapper} ref={wrapperRef}>
            {!bare && <label>{field.label}</label>}
            <input
                ref={inputRef}
                type="text"
                name={field.field + "__search"}
                autoComplete="off"
                value={search}
                onChange={e => {
                    setSearch(e.target.value);
                    setOpen(true);
                }}
                onFocus={() => setOpen(true)}
                onKeyDown={handleKeyDown}
                className={styles.autocompleteInput}
            />
            <input type="hidden" name={field.field} value={selectedId} required={isRequired} />
            {isLoading && <div className={styles.autocompleteLoading}>Загрузка...</div>}
            {open && !isLoading && (
                <div className={styles.autocompleteDropdown}>
                    {filtered.length === 0 ? (
                        <div className={styles.autocompleteEmpty}>Ничего не найдено</div>
                    ) : (
                        filtered.map(opt => (
                            <div
                                key={opt.id}
                                className={`${styles.autocompleteOption} ${String(opt.id) === String(selectedId) ? styles.autocompleteOptionSelected : ""}`}
                                onClick={() => handleSelect(opt)}
                                onMouseDown={e => e.preventDefault()}
                            >
                                {opt.name}
                            </div>
                        ))
                    )}
                </div>
            )}
        </div>
    );
};

// Клиентский пред-чек размера (источник истины — backend FILE_MAX_MB, там же
// итоговая валидация; сообщение выводим из той же константы, чтобы не врать).
const MAX_FILE_SIZE = 25 * 1024 * 1024;

const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} Б`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`;
    return `${(bytes / 1024 / 1024).toFixed(1)} МБ`;
};

const asFileId = (value: unknown): string =>
    value === null || value === undefined || value === "" ? "" : String(value);

interface AttachedFile {
    id: number | string;
    file_name: string;
    mime: string;
    size: number;
    tag?: string;
}

interface FreshFile {
    id: string;
    name: string;
    size: number;
}

const pluralFiles = (n: number): string => {
    const m10 = n % 10;
    const m100 = n % 100;
    if (m10 === 1 && m100 !== 11) return "файл";
    if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return "файла";
    return "файлов";
};

// FileField — выборка файлов роли tag (дефолт — имя поля), а не указатель
// на один файл. Документ таблицы файлов не знает: поле виртуальное, в строку
// не пишется; пайплайн сверяет точное множество (attach выбранных — шаг
// привязки, detach снятых — отдельный шаг с <имя>_detached).
// Крестик НЕ удаляет строку files: снятый остаётся сиротой, его подбирает
// GC. Поэтому список строится только из действий пользователя, а не из
// «всего, чего нет»: несвежий/упавший список не сносит чужие привязки.
const FileField = ({field, value, isEditing, bare, serviceRef, docId}: FieldRowProps) => {
    void value;
    const tag = field.tag ?? field.field;
    // Файлы документа с этой ролью (сервер — истина для существующих).
    const [serverFiles, setServerFiles] = useState<AttachedFile[] | null>(null);
    const [listFailed, setListFailed] = useState(false);
    const [listRetry, setListRetry] = useState(0);
    // Свежие загрузки этой сессии (сироты до сохранения).
    const [fresh, setFresh] = useState<FreshFile[]>([]);
    // Снятые с выборки (крестик): уйдут шагом detach, строкой не удаляются.
    const [detached, setDetached] = useState<string[]>([]);
    const [uploading, setUploading] = useState(false);
    const [errorMsg, setErrorMsg] = useState("");
    const [isDragging, setIsDragging] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const isRequired = field.required ?? false;
    const maxCount =
        typeof field.max_count === "number" && field.max_count > 0 ? Math.floor(field.max_count) : undefined;
    const canList =
        !!serviceRef && docId !== null && docId !== undefined && docId !== "";


    // Файлы своей роли одним запросом (не N по файлам). Нужен и в просмотре
    // (карточки), и в редактировании (истина для инициализации выборки).
    useEffect(() => {
        if (!canList || !serviceRef) {
            setServerFiles(null);
            return;
        }
        let cancelled = false;
        setServerFiles(null);
        setListFailed(false);
        api.get<{ data?: AttachedFile[] } | AttachedFile[]>(
            `/api/${serviceRef}/files?id_document=${encodeURIComponent(String(docId))}&tag=${encodeURIComponent(tag)}`
        ).then(
            (resp) => {
                if (cancelled) return;
                const raw = resp as unknown as { data?: unknown };
                const list = Array.isArray(raw) ? raw : Array.isArray(raw?.data) ? raw.data as AttachedFile[] : [];
                setServerFiles(list.filter((f) => f && f.id !== null && f.id !== undefined));
            },
            () => {
                if (!cancelled) setListFailed(true);
            },
        );
        return () => {
            cancelled = true;
        };
    }, [serviceRef, docId, canList, tag, listRetry]);

    const detachedSet = useMemo(() => new Set(detached), [detached]);
    // Выборка: серверные минус снятые + свежие. Именно она едет скрытыми
    // инпутами шагу привязки; снятые — отдельным списком шагу detach.
    const activeServer = useMemo(
        () => (serverFiles ?? []).filter((f) => !detachedSet.has(String(f.id))),
        [serverFiles, detachedSet],
    );
    const selectedIds = useMemo(
        () => [...activeServer.map((f) => String(f.id)), ...fresh.map((f) => f.id)],
        [activeServer, fresh],
    );
    const limitReached = maxCount !== undefined && selectedIds.length >= maxCount;

    const fileUrl = (id: string | number | unknown): string | null =>
        serviceRef && id !== null && id !== undefined && id !== ""
            ? `/api/${serviceRef}/files?content=${encodeURIComponent(String(id))}`
            : null;

    const unselectServer = useCallback((id: string | number) => {
        const key = String(id);
        setDetached((prev) => (prev.includes(key) ? prev : [...prev, key]));
    }, []);

    const unselectFresh = useCallback((id: string | number) => {
        // Свежая загрузка: просто убираем из выборки (сирота, подберёт GC)
        // и фиксируем в detached-списке для единообразия шага detach.
        const key = String(id);
        setFresh((prev) => prev.filter((f) => f.id !== key));
        setDetached((prev) => (prev.includes(key) ? prev : [...prev, key]));
    }, []);

    const renderFileCards = (files: AttachedFile[], opts?: {onRemove?: (id: string | number) => void; freshMark?: boolean}) => (
        <ul className={styles.fileList}>
            {files.map((f) => {
                const url = fileUrl(f.id);
                const inner = (
                    <>
                        <span className={styles.fileListName}>
                            {f.file_name || `Файл ${f.id}`}
                        </span>
                    {typeof f.size === "number" && Number.isFinite(f.size) && (
                        <span className={styles.fileListSize}>{formatFileSize(f.size)}</span>
                    )}
                    {opts?.freshMark && <span className={styles.fileListSize}>новый</span>}
                    {opts?.onRemove && (
                        <button
                            type="button"
                            className={styles.filePickBtn}
                            title="Убрать из выборки (файл не удаляется)"
                            onClick={() => opts.onRemove!(f.id)}
                        >
                            ×
                        </button>
                    )}
                    </>
                );
                return (
                    <li key={String(f.id)}>
                        {url ? (
                            <a className={styles.fileCard} href={url} target="_blank" rel="noreferrer">
                                {inner}
                            </a>
                        ) : (
                            <span className={styles.fileCard}>{inner}</span>
                        )}
                    </li>
                );
            })}
        </ul>
    );

    if (!isEditing) {
        // Просмотр: серверный список своей роли; висячих id в поле больше
        // нет (поле виртуальное), придумывать карточки не из чего.
        let body: ReactNode;
        if (canList && serverFiles === null && !listFailed) {
            body = <span>…</span>;
        } else if (serverFiles !== null && serverFiles.length > 0) {
            body = renderFileCards(serverFiles);
        } else {
            body = <span>—</span>;
        }
        return (
            <>
                {!bare && <label>{field.label}</label>}
                <div className={styles.viewValue}>{body}</div>
            </>
        );
    }

    const uploadFile = (file: File) => {
        if (!serviceRef) {
            setErrorMsg("Неизвестная сущность");
            return;
        }
        if (limitReached) {
            setErrorMsg(`Максимум: ${maxCount} ${pluralFiles(maxCount ?? 0)} — уберите лишний, чтобы загрузить другой`);
            return;
        }
        if (file.size > MAX_FILE_SIZE) {
            setErrorMsg(`Файл больше ${formatFileSize(MAX_FILE_SIZE)}`);
            return;
        }
        setUploading(true);
        setErrorMsg("");
        const reader = new FileReader();
        reader.onload = () => {
            void (async () => {
                try {
                    const result = String(reader.result ?? "");
                    const base64 = result.includes(",") ? result.slice(result.indexOf(",") + 1) : result;
                    const resp = await api.post<Record<string, unknown>>(
                        `/api/${serviceRef}/files`,
                        {file_name: file.name, data: base64, tag},
                    );
                    const rec = resp as Record<string, unknown>;
                    const wrapped = rec.data as Record<string, unknown> | undefined;
                    const rawId = rec.id ?? wrapped?.id;
                    if (rawId === null || rawId === undefined || rawId === "") {
                        throw new Error("empty id");
                    }
                    const newId = String(rawId);
                    setFresh((prev) => [...prev, {id: newId, name: file.name, size: file.size}]);
                    // Перевыбор ранее снятого: снимаем пометку detach.
                    setDetached((prev) => prev.filter((d) => d !== newId));
                } catch {
                    setErrorMsg("Не удалось загрузить файл");
                } finally {
                    setUploading(false);
                }
            })();
        };
        reader.onerror = () => {
            setUploading(false);
            setErrorMsg("Не удалось прочитать файл");
        };
        reader.readAsDataURL(file);
    };

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        e.target.value = "";
        if (!file) return;
        uploadFile(file);
    };

    return (
        <>
            {!bare && (
                <label>
                    {field.label} {isRequired && <span style={{color: "red"}}>*</span>}
                </label>
            )}
            {listFailed && (
                <div style={{color: "red", fontSize: "12px"}}>
                    Не удалось загрузить список файлов{" "}
                    <button type="button" className={styles.filePickBtn} onClick={() => setListRetry((n) => n + 1)}>
                        Повторить
                    </button>
                </div>
            )}
            {activeServer.length > 0 && (
                <div className={styles.viewValue}>{renderFileCards(activeServer, {onRemove: unselectServer})}</div>
            )}
            {fresh.length > 0 && (
                <div className={styles.viewValue}>
                    {renderFileCards(
                        fresh.map((f) => ({id: f.id, file_name: f.name, mime: "", size: f.size})),
                        {onRemove: unselectFresh, freshMark: true},
                    )}
                </div>
            )}
            {!limitReached && (
            <div
                className={`${styles.fileDropZone} ${isDragging ? styles.fileDropZoneActive : ""}`}
                onDragOver={(e) => {
                    e.preventDefault();
                    setIsDragging(true);
                }}
                onDragLeave={() => setIsDragging(false)}
                onDrop={(e) => {
                    e.preventDefault();
                    setIsDragging(false);
                    const file = e.dataTransfer.files?.[0];
                    if (file) uploadFile(file);
                }}
            >
                <input
                    ref={fileInputRef}
                    type="file"
                    accept=".png,.jpg,.jpeg,.pdf"
                    onChange={handleChange}
                    className={styles.fileInputHidden}
                />
                <button
                    type="button"
                    className={styles.filePickBtn}
                    onClick={() => fileInputRef.current?.click()}
                >
                    Выбрать файл
                </button>
                {maxCount !== undefined && (
                    <div className={styles.fileHint}>
                        Максимум: {maxCount} {pluralFiles(maxCount)}
                    </div>
                )}
                {uploading && <div className={styles.autocompleteLoading}>Загрузка...</div>}
                {errorMsg && (
                    <div style={{color: "red", fontSize: "12px"}}>{errorMsg}</div>
                )}
            </div>
            )}
            {limitReached && maxCount !== undefined && (
                <div className={styles.fileHint}>
                    Максимум: {maxCount} {pluralFiles(maxCount)} — уберите лишний, чтобы загрузить другой
                </div>
            )}
            {selectedIds.map((id) => (
                <input key={`sel-${id}`} type="hidden" name={field.field} value={id} />
            ))}
            {detached.map((id) => (
                <input key={`det-${id}`} type="hidden" name={`${field.field}${FILE_DETACHED_SUFFIX}`} value={id} />
            ))}
        </>
    );
};
