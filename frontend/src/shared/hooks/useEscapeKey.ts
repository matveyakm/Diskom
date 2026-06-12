import { useEffect } from "react";

export function useEscapeKey(callback: () => void, enabled: boolean = true) {
    useEffect(() => {
        if (!enabled) return;

        const handleKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") callback();
        };

        document.addEventListener("keydown", handleKey);
        return () => document.removeEventListener("keydown", handleKey);
    }, [enabled, callback]);
}
