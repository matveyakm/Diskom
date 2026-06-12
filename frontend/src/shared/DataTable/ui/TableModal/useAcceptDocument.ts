import {useMutation, useQueryClient} from "@tanstack/react-query";
import {tableApi} from "@/shared/api/fetchTableData";
import {TableRow} from "@/shared/DataTable/types";
import {FormField} from "@/entities/document/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import {parseActionConfig, processAction} from "@/shared/api/actionPipeline";

type UseAcceptDocumentProps<T extends TableRow = TableRow> = {
    docName: string;
    selectedRow: T | null;
    fields: FormField[];
    tableMeta?: MenuItem;
    onSuccess?: () => void;
    onError?: () => void;
};

export const useAcceptDocument = <T extends TableRow = TableRow>({
    docName,
    selectedRow,
    fields,
    tableMeta,
    onSuccess,
    onError,
}: UseAcceptDocumentProps<T>) => {
    const queryClient = useQueryClient();

    const acceptRowMutation = async (): Promise<void> => {
        const pipelineConfig = tableMeta?.on_acceptance;

        if (!pipelineConfig) {
            if (!selectedRow) throw new Error("Нет данных для проведения");
            const id = selectedRow.id;
            if (!id) throw new Error("Не удалось найти ID записи");
            await tableApi(docName).update({id, status: 2});
            return;
        }
        if (!parseActionConfig(pipelineConfig)?.length) {
            throw new Error("Некорректная конфигурация проведения");
        }

        await processAction(
            "onAcceptance",
            {...selectedRow, id: selectedRow?.id} as Record<string, unknown>,
            pipelineConfig,
            fields,
            tableMeta?.id,
        );
    };

    return useMutation({
        mutationFn: acceptRowMutation,
        onSuccess: () => {
            queryClient.invalidateQueries({queryKey: ["tableData", docName]});
            queryClient.invalidateQueries({queryKey: ["reference"]});
            onSuccess?.();
        },
        onError: () => {
            onError?.();
        },
    });
};
