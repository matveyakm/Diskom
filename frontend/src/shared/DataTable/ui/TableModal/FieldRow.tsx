import {useQuery} from "@tanstack/react-query";
import {tableApi} from "@/shared/api/fetchTableData";
import {fetchAllPages} from "@/shared/api/fetchAllPages";
import {FieldRowProps} from "./types";
import styles from "./TableModal.module.scss";
import {useEffect, useState, useRef, useCallback} from "react";
import {isDateField, formatDateForDisplay, toDateInputValue, getTodayDate} from "./utils";

export const FieldRow = ({field, value, isEditing, error}: FieldRowProps) => {
    const [currentValue, setCurrentValue] = useState(String(value ?? ""));

    useEffect(() => {
        setCurrentValue(String(value ?? ""));
    }, [value]);

    if (field.type === "select" && field.end_point) {
        return <SelectField field={field} value={value} isEditing={isEditing} error={error}/>;
    }

    if (isDateField(field)) {
        return <DateField field={field} value={value} isEditing={isEditing} error={error}/>;
    }

    const isRequired = field.required ?? false;

    return (
        <>
            <label>
                {field.label} {isRequired && <span style={{color: "red"}}>*</span>}
            </label>
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

const DateField = ({field, value, isEditing}: FieldRowProps) => {
    const [currentValue, setCurrentValue] = useState(() => {
        if (value) return toDateInputValue(value);
        if (isEditing) return getTodayDate();
        return "";
    });

    const isRequired = field.required ?? false;

    return (
        <>
            <label>
                {field.label} {isRequired && <span style={{color: "red"}}>*</span>}
            </label>
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

interface SelectOption {
    id: string | number;
    name: string;
}

const SelectField = ({field, value, isEditing}: FieldRowProps) => {
    const [search, setSearch] = useState("");
    const [open, setOpen] = useState(false);
    const [selectedId, setSelectedId] = useState<string | number>(() =>
        String(value ?? "")
    );
    const wrapperRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    const isRequired = field.required ?? false;

    const {data: allOptions, isLoading, error: queryError} = useQuery({
        queryKey: ["reference", field.end_point],
        queryFn: () =>
            fetchAllPages<SelectOption>(
                (page, limit) =>
                    tableApi(field.end_point!).getData({page, limit}) as Promise<{
                        data: SelectOption[];
                        pagination: {total_pages: number};
                    }>,
                100,
            ),
        enabled: !!field.end_point,
    });

    const typedOptions: SelectOption[] = allOptions ?? [];

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

    const handleSelect = useCallback((opt: SelectOption) => {
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
                <label>{field.label}</label>
                <div style={{color: "red", fontSize: "12px"}}>Ошибка загрузки справочника</div>
            </>
        );
    }

    if (!isEditing) {
        const selectedOption = typedOptions.find(o => String(o.id) === String(value));
        const displayValue = selectedOption ? selectedOption.name : String(value ?? "—");
        return (
            <>
                <label>{field.label}</label>
                <div className={styles.viewValue}>{displayValue}</div>
            </>
        );
    }

    return (
        <div className={styles.autocompleteWrapper} ref={wrapperRef}>
            <label>{field.label}</label>
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
