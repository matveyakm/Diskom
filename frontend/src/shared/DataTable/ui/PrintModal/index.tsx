import {createPortal} from "react-dom";
import {useEffect, useState} from "react";
import styles from "./PrintModal.module.scss";
import {buildPrint, downloadPrint, PrintFormat} from "@/shared/api/printApi";
import {describeActionError} from "@/shared/api/actionPipeline";

interface PrintModalProps {
    docName: string | undefined;
    rowId: number | string;
    docTitle: string;
    onClose: () => void;
}

type Phase = "select" | "building" | "ready" | "error";

// Модалка печати: выбор формата → "Распечатать" (POST ?print= собирает
// файл на бэкенде) → иконка файла с размером → клик скачивает
// (PDF открывается в новой вкладке, DOCX — через download).
// Одноразовый blob-URL чистится при закрытии/размонтировании.
export const PrintModal = ({docName, rowId, docTitle, onClose}: PrintModalProps) => {
    const [format, setFormat] = useState<PrintFormat>("pdf");
    const [phase, setPhase] = useState<Phase>("select");
    const [meta, setMeta] = useState<{ token: string; size: number; filename: string } | null>(null);
    const [error, setError] = useState("");
    const [blobUrl, setBlobUrl] = useState<string | null>(null);

    useEffect(() => {
        return () => {
            if (blobUrl) URL.revokeObjectURL(blobUrl);
        };
    }, [blobUrl]);

    const handleBuild = async () => {
        if (!docName) return;
        setPhase("building");
        setError("");
        try {
            const m = await buildPrint(docName, format, rowId);
            setMeta({token: m.token, size: m.size, filename: m.filename});
            setPhase("ready");
        } catch (err) {
            setError(`Не удалось собрать документ (${describeActionError(err)}).`);
            setPhase("error");
        }
    };

    const handleDownload = async () => {
        if (!docName || !meta) return;
        try {
            const blob = await downloadPrint(docName, meta.token);
            const url = URL.createObjectURL(blob);
            setBlobUrl(url);
            if (format === "pdf") {
                window.open(url, "_blank");
            } else {
                const a = document.createElement("a");
                a.href = url;
                a.download = meta.filename;
                document.body.appendChild(a);
                a.click();
                a.remove();
            }
        } catch (err) {
            setError(`Не удалось скачать файл (${describeActionError(err)}).`);
            setPhase("error");
        }
    };

    const formatSize = (bytes: number): string => {
        if (bytes < 1024) return `${bytes} Б`;
        return `${(bytes / 1024).toFixed(1)} КБ`;
    };

    return createPortal(
        <div className={styles.overlay} onClick={onClose}>
            <div className={styles.window} onClick={(e) => e.stopPropagation()}>
                <header className={styles.header}>
                    <h3 className={styles.title}>Печать: {docTitle}</h3>
                    <button className={styles.closeX} onClick={onClose} aria-label="Закрыть">
                        ✕
                    </button>
                </header>

                <div className={styles.body}>
                    {(phase === "select" || phase === "building") && (
                        <>
                            <div className={styles.formats}>
                                {(["pdf", "docx"] as PrintFormat[]).map(f => (
                                    <label key={f} className={styles.formatOption}>
                                        <input
                                            type="radio"
                                            name="print-format"
                                            checked={format === f}
                                            disabled={phase === "building"}
                                            onChange={() => setFormat(f)}
                                        />
                                        <span className={styles.formatBadge}>{f.toUpperCase()}</span>
                                        <span className={styles.formatHint}>
                                            {f === "pdf" ? "откроется в новой вкладке" : "скачается файлом"}
                                        </span>
                                    </label>
                                ))}
                            </div>
                            <button type="button" onClick={handleBuild}
                                    className={styles.printBtn}
                                    disabled={phase === "building"}>
                                {phase === "building" ? "Собираем…" : "Распечатать"}
                            </button>
                        </>
                    )}

                    {phase === "ready" && meta && (
                        <button type="button" onClick={handleDownload} className={styles.fileRow}
                                title="Скачать">
                            <span className={styles.fileBadge}>{format.toUpperCase()}</span>
                            <span className={styles.fileName}>{meta.filename}</span>
                            <span className={styles.fileSize}>{formatSize(meta.size)}</span>
                        </button>
                    )}

                    {phase === "error" && (
                        <>
                            <div className={styles.errorText}>{error}</div>
                            <button type="button" onClick={() => setPhase("select")}
                                    className={styles.printBtn}>
                                Назад
                            </button>
                        </>
                    )}
                </div>
            </div>
        </div>,
        document.body,
    );
};
