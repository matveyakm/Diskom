import {useMutation, useQueryClient} from "@tanstack/react-query";
import {tableApi} from "@/shared/api/fetchTableData";
import {FormField} from "@/entities/document/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import {processAction} from "@/shared/api/actionPipeline";

type UseSaveDocumentProps = {
    docName: string;
    mode: "edit" | "create";
    visibleFields: FormField[];
    tableMeta?: MenuItem;
};

export const useSaveDocument = ({
    docName,
    mode,
    visibleFields,
    tableMeta,
}: UseSaveDocumentProps) => {
    const queryClient = useQueryClient();

    const saveViaPipeline = async (payload: Record<string, unknown>): Promise<Record<string, unknown>> => {
        const pipelineConfig = mode === "create"
            ? tableMeta?.on_save_new
            : tableMeta?.on_save_edit;

        if (pipelineConfig) {
            return await processAction(
                mode === "create" ? "onSaveNew" : "onSaveEdit",
                payload,
                pipelineConfig,
                visibleFields,
                tableMeta?.id,
            );
        }
        if (mode === "create") {
            return (await tableApi(docName).create(payload)) as Record<string, unknown>;
        }
        return (await tableApi(docName).update(payload)) as Record<string, unknown>;
    };

    return useMutation({
        mutationFn: saveViaPipeline,
        onSuccess: () => {
            queryClient.invalidateQueries({queryKey: ["tableData", docName]});
            queryClient.invalidateQueries({queryKey: ["reference"]});
        },
    });
};
