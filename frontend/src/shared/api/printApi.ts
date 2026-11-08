import {api} from "@/shared/api/AxiosClient";

export type PrintFormat = "pdf" | "docx";

export interface PrintMeta {
    token: string;
    size: number;
    filename: string;
    mime: string;
}

/** Одна несмертельная проблема сборки (бэкенд: collectPrintProblems).
 * kind: field — пустое обязательное поле, image/slot/table — см. Message. */
export interface PrintProblem {
    kind: string;
    ref: string;
    label?: string;
    message: string;
}

/** 500 от сборки PDF: {error, details}. DOCX вместо этого отдаёт файл
 * с красными ОШИБКА/квадратами прямо в документе. */
export interface PrintBuildFailure {
    error: string;
    details?: PrintProblem[];
}

/** Вытаскивает структурированные проблемы из axios-ошибки сборки. */
export const extractPrintProblems = (err: unknown): PrintProblem[] | null => {
    const data = (err as { response?: { data?: unknown } })?.response?.data as PrintBuildFailure | undefined;
    if (data && Array.isArray(data.details) && data.details.length > 0) {
        return data.details;
    }
    return null;
};

// Двухшаговая выдача (см. source/shared/controller/print_handler.go):
// 1. POST ?print=pdf|docx {id} → метаданные готового файла (байты лежат
//    в store сервиса 5 минут), 2. GET ?print=file&token= → blob.
// Синхронной выдачи одним запросом нет осознанно: между "Распечатать"
// и скачиванием пользователь видит иконку файла с размером.
export const buildPrint = async (
    tableName: string,
    format: PrintFormat,
    id: number | string,
): Promise<PrintMeta> => {
    return await api.post<PrintMeta>(`/api/${tableName}?print=${format}`, {id});
};

export const downloadPrint = async (tableName: string, token: string): Promise<Blob> => {
    return await api.get<Blob>(`/api/${tableName}?print=file&token=${encodeURIComponent(token)}`, {
        responseType: "blob",
    });
};
