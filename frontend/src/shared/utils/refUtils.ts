import { type RefObject } from "react";

export function asElementRef(ref: RefObject<HTMLDivElement | null>): RefObject<HTMLElement> {
    return ref as unknown as RefObject<HTMLElement>;
}

export function asElementRefs(refs: RefObject<HTMLDivElement | null>[]): RefObject<HTMLElement>[] {
    return refs as unknown as RefObject<HTMLElement>[];
}
