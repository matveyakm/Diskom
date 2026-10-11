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
};

export const useAcceptDocument = <T extends TableRow = TableRow>({
    docName,
    selectedRow,
    fields,
    tableMeta,
}: UseAcceptDocumentProps<T>) => {
    const queryClient = useQueryClient();

    // Accepts either the opened row (toolbar "Провести") or an explicitly
    // passed one ("Сохранить и провести" right after creation).
    const acceptRowMutation = async (rowOverride?: T | null): Promise<void> => {
        const row = rowOverride ?? selectedRow;
        const pipelineConfig = tableMeta?.on_acceptance;

        if (!pipelineConfig) {
            if (!row) throw new Error("Нет данных для проведения");
            const id = row.id;
            if (!id) throw new Error("Не удалось найти ID записи");
            await tableApi(docName).update({id, status: 2});
            return;
        }
        if (!parseActionConfig(pipelineConfig)?.length) {
            throw new Error("Некорректная конфигурация проведения");
        }

        await processAction(
            "onAcceptance",
            {...row, id: row?.id} as Record<string, unknown>,
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
        },
    });
};
