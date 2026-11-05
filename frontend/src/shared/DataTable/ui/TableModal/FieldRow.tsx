import {useReferenceOptions, ReferenceOption} from "@/shared/api/useReferenceOptions";
import {api} from "@/shared/api/AxiosClient";
import {FieldRowProps} from "./types";
import styles from "./TableModal.module.scss";
import {useEffect, useState, useRef, useCallback, useMemo} from "react";
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

type FileUploadStatus = "idle" | "uploading" | "done" | "error";

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
}

const FileField = ({field, value, isEditing, bare, serviceRef, docId}: FieldRowProps) => {
    const [fileId, setFileId] = useState(() => asFileId(value));
    const [fileName, setFileName] = useState("");
    const [fileSize, setFileSize] = useState<number | null>(null);
    const [status, setStatus] = useState<FileUploadStatus>("idle");
    const [errorMsg, setErrorMsg] = useState("");
    const [isDragging, setIsDragging] = useState(false);
    const [attached, setAttached] = useState<AttachedFile[] | null>(null);
    const [listFailed, setListFailed] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const isRequired = field.required ?? false;

    useEffect(() => {
        setFileId(asFileId(value));
    }, [value]);

    // Просмотр: явный список прикреплённых файлов с метаданными
    // (один запрос по документу, а не N по файлам).
    useEffect(() => {
        if (isEditing) return;
        if (!serviceRef || docId === null || docId === undefined || docId === "") {
            setAttached(null);
            return;
        }
        let cancelled = false;
        setAttached(null);
        setListFailed(false);
        api.get<{ data?: AttachedFile[] } | AttachedFile[]>(
            `/api/${serviceRef}/files?id_document=${encodeURIComponent(String(docId))}`
        ).then(
            (resp) => {
                if (cancelled) return;
                const raw = resp as unknown as { data?: unknown };
                const list = Array.isArray(raw) ? raw : Array.isArray(raw?.data) ? raw.data as AttachedFile[] : [];
                setAttached(list.filter((f) => f && f.id !== null && f.id !== undefined));
            },
            () => {
                if (!cancelled) setListFailed(true);
            },
        );
        return () => {
            cancelled = true;
        };
    }, [isEditing, serviceRef, docId]);

    const downloadUrl =
        fileId && serviceRef
            ? `/api/${serviceRef}/files?content=${encodeURIComponent(fileId)}`
            : null;

    if (!isEditing) {
        return (
            <>
                {!bare && <label>{field.label}</label>}
                <div className={styles.viewValue}>
                    {attached === null && !listFailed ? (
                        <span>…</span>
                    ) : attached !== null && attached.length > 0 ? (
                        <ul className={styles.fileList}>
                            {attached.map((f) => (
                                <li key={String(f.id)} className={styles.fileListItem}>
                                    <span className={styles.fileListName}>{f.file_name || `Файл ${f.id}`}</span>
                                    {typeof f.size === "number" && (
                                        <span className={styles.fileListSize}>{formatFileSize(f.size)}</span>
                                    )}
                                    {serviceRef && (
                                        <a
                                            href={`/api/${serviceRef}/files?content=${encodeURIComponent(String(f.id))}`}
                                            target="_blank"
                                            rel="noreferrer"
                                        >
                                            Скачать
                                        </a>
                                    )}
                                </li>
                            ))}
                        </ul>
                    ) : listFailed && downloadUrl ? (
                        <a href={downloadUrl} target="_blank" rel="noreferrer">Скачать</a>
                    ) : (
                        <span>—</span>
                    )}
                </div>
            </>
        );
    }

    const uploadFile = (file: File) => {
        if (!serviceRef) {
            setStatus("error");
            setErrorMsg("Неизвестная сущность");
            return;
        }
        if (file.size > MAX_FILE_SIZE) {
            setStatus("error");
            setErrorMsg("Файл больше 25 МБ");
            return;
        }
        setFileName(file.name);
        setFileSize(file.size);
        setStatus("uploading");
        setErrorMsg("");
        const reader = new FileReader();
        reader.onload = () => {
            void (async () => {
                try {
                    const result = String(reader.result ?? "");
                    const base64 = result.includes(",") ? result.slice(result.indexOf(",") + 1) : result;
                    const resp = await api.post<Record<string, unknown>>(
                        `/api/${serviceRef}/files`,
                        {file_name: file.name, data: base64},
                    );
                    const rec = resp as Record<string, unknown>;
                    const wrapped = rec.data as Record<string, unknown> | undefined;
                    const rawId = rec.id ?? wrapped?.id;
                    if (rawId === null || rawId === undefined || rawId === "") {
                        throw new Error("empty id");
                    }
                    setFileId(String(rawId));
                    setStatus("done");
                } catch {
                    setStatus("error");
                    setErrorMsg("Не удалось загрузить файл");
                }
            })();
        };
        reader.onerror = () => {
            setStatus("error");
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
            {fileId && status !== "uploading" && downloadUrl && (
                <div className={styles.viewValue}>
                    <a href={downloadUrl} target="_blank" rel="noreferrer">Скачать</a>
                </div>
            )}
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
                {fileName && (
                    <div className={styles.autocompleteLoading}>
                        {fileName}{fileSize !== null ? ` (${formatFileSize(fileSize)})` : ""}
                    </div>
                )}
                {status === "uploading" && <div className={styles.autocompleteLoading}>Загрузка...</div>}
                {status === "done" && <div className={styles.autocompleteLoading}>Загружено</div>}
                {status === "error" && errorMsg && (
                    <div style={{color: "red", fontSize: "12px"}}>{errorMsg}</div>
                )}
            </div>
            <input type="hidden" name={field.field} value={fileId} required={isRequired}/>
        </>
    );
};
