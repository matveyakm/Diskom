export async function fetchAllPages<T>(
    fetcher: (page: number, limit: number) => Promise<{data: T[]; pagination: {total_pages: number}}>,
    limit = 100,
): Promise<T[]> {
    const first = await fetcher(1, limit);
    const totalPages = first.pagination.total_pages;
    const allData: T[] = [...first.data];

    const remaining = Array.from({length: totalPages - 1}, (_, i) => i + 2);
    const results = await Promise.all(remaining.map(page => fetcher(page, limit)));
    for (const r of results) {
        allData.push(...r.data);
    }

    return allData;
}
