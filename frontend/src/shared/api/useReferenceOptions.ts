import {useQuery} from "@tanstack/react-query";
import {tableApi} from "@/shared/api/fetchTableData";
import {fetchAllPages} from "@/shared/api/fetchAllPages";

export interface ReferenceOption {
    id: string | number;
    name: string;
}

// Общий источник вариантов справочника: модалка (селекты формы) и
// Excel-попап фильтров (имена для галочек). Один кеш ["reference", end_point].
export const useReferenceOptions = (endPoint: string | undefined) => {
    return useQuery({
        queryKey: ["reference", endPoint],
        queryFn: () =>
            fetchAllPages<ReferenceOption>(
                (page, limit) =>
                    tableApi(endPoint!).getData({page, limit}) as Promise<{
                        data: ReferenceOption[];
                        pagination: {total_pages: number};
                    }>,
                100,
            ),
        enabled: !!endPoint,
    });
};
