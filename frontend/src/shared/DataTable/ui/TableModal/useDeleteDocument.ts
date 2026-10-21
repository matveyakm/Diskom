import {useMutation, useQueryClient} from "@tanstack/react-query";
import {tableApi} from "@/shared/api/fetchTableData";
import {TableRow} from "@/shared/DataTable/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import {processAction} from "@/shared/api/actionPipeline";

type UseDeleteDocumentProps<T extends TableRow = TableRow> = {
    docName: string;
    selectedRow: T | null;
    tableMeta?: MenuItem;
    onSuccess?: () => void;
    onError?: (err: unknown) => void;
};

export const useDeleteDocument = <T extends TableRow = TableRow>({
    docName,
    selectedRow,
    tableMeta,
    onSuccess,
    onError,
}: UseDeleteDocumentProps<T>) => {
    const queryClient = useQueryClient();

    const deleteRowMutation = async () => {
        if (!selectedRow) throw new Error("Нет данных для удаления");
        const id = selectedRow.id;
        if (!id) throw new Error("Не удалось найти ID записи для удаления");

        const pipelineConfig = tableMeta?.on_delete;
        if (pipelineConfig) {
            await processAction(
                "onDelete",
                {...selectedRow} as Record<string, unknown>,
                pipelineConfig,
                [],
                tableMeta?.id,
            );
        } else {
            await tableApi(docName).delete(id);
        }
    };

    return useMutation({
        mutationFn: deleteRowMutation,
        onSuccess: () => {
            // См. комментарий в useSaveDocument: удаление через пайплайн трогает чужие таблицы.
            queryClient.invalidateQueries({queryKey: ["tableData"]});
            queryClient.invalidateQueries({queryKey: ["reference"]});
            onSuccess?.();
        },
        onError: (err) => {
            onError?.(err);
        },
    });
};
