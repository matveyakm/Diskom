import {api} from "@/shared/api/AxiosClient";

export type PrintFormat = "pdf" | "docx";

export interface PrintMeta {
    token: string;
    size: number;
    filename: string;
    mime: string;
}

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
