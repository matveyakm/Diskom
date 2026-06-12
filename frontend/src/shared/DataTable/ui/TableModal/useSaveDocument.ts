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
    onSuccess?: () => void;
    onError?: () => void;
};

export const useSaveDocument = ({
    docName,
    mode,
    visibleFields,
    tableMeta,
    onSuccess,
    onError,
}: UseSaveDocumentProps) => {
    const queryClient = useQueryClient();

    const saveViaPipeline = async (payload: Record<string, unknown>): Promise<void> => {
        const pipelineConfig = mode === "create"
            ? tableMeta?.on_save_new
            : tableMeta?.on_save_edit;

        if (pipelineConfig) {
            await processAction(
                mode === "create" ? "onSaveNew" : "onSaveEdit",
                payload,
                pipelineConfig,
                visibleFields,
                tableMeta?.id,
            );
        } else {
            if (mode === "create") {
                await tableApi(docName).create(payload);
            } else {
                await tableApi(docName).update(payload);
            }
        }
    };

    return useMutation({
        mutationFn: saveViaPipeline,
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
