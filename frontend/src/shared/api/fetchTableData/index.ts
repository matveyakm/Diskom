import {api} from "@/shared/api/AxiosClient";
import {useQuery} from "@tanstack/react-query";
import type {PaginatedResponse} from "@/shared/api/fetchTableData/types";
import {DirectoryItem, DocumentItem} from "@/shared/api/fetchTableData/types";

interface GetDataParams {
    page?: number;
    limit?: number;
}

export const tableApi = (tableName: string) => {
    const url = `/api/${tableName}`;

    return {
        getData: async (params?: GetDataParams): Promise<PaginatedResponse<DirectoryItem | DocumentItem>> => {
            const queryParams: Record<string, string> = {};
            if (params?.page) queryParams.page = String(params.page);
            if (params?.limit) queryParams.limit = String(params.limit);

            return await api.get<PaginatedResponse<DirectoryItem | DocumentItem>>(url, {params: queryParams});
        },

        getById: async (id: number | string) => {
            return api.get(`${url}/${id}`);
        },

        create: async (data: unknown) => {
            return api.post(url, data);
        },

        update: async (data: unknown) => {
            const id = (data as Record<string, unknown>).id;
            const query = id != null ? `?id=${encodeURIComponent(String(id))}` : "";
            return api.put(`${url}${query}`, data);
        },

        patch: async (data: unknown) => {
            const id = (data as Record<string, unknown>).id;
            const query = id != null ? `?id=${encodeURIComponent(String(id))}` : "";
            return api.patch(`${url}${query}`, data);
        },

        delete: async (id: number | string) => {
            await api.delete(`${url}?id=${encodeURIComponent(String(id))}`);
        },
    };
};

export const useTableData = (docName: string, page = 1, limit = 50) => {
    return useQuery({
        queryKey: ["tableData", docName, page, limit],
        queryFn: () => tableApi(docName).getData({page, limit}),
        enabled: !!docName,
    });
};
