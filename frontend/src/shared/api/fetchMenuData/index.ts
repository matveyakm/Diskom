import {api} from "@/shared/api/AxiosClient";
import {useQuery} from "@tanstack/react-query";
import {MenuItem} from "@/shared/api/fetchMenuData/types";

const fetchMenuData = async () => {
    const response = await api.get("/api/system_object_directory", {params: {limit: "100"}}) as { data: MenuItem[] };
    const item = response.data.find((i: MenuItem) => i.id === 12);
    if (item) {
        console.log("[menuData] raw item 12:", JSON.parse(JSON.stringify(item)));
        console.log("[menuData] keys:", Object.keys(item));
    }
    return response.data.sort((a, b) => (a.output_order ?? 0) - (b.output_order ?? 0));
};

const useMenuData = () => {
    return useQuery({
        queryKey: ["menuData"],
        queryFn: () => fetchMenuData(),
    });
};

if (typeof window !== "undefined") {
    (window as unknown as Record<string, unknown>).__debugMenu = async () => {
        const raw = await fetch("/api/system_object_directory?limit=100");
        const json = await raw.json();
        const item = json.data?.find((i: MenuItem) => i.id === 12);
        console.log("[__debugMenu] raw item 12:", item);
        console.log("[__debugMenu] keys:", Object.keys(item ?? {}));
        return item;
    };
}

export default useMenuData;
