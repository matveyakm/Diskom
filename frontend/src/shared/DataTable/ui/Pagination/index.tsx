import styles from "./PaginationContainer.module.scss";

interface PaginationProps {
    pageIndex: number;
    pageCount: number;
    onPageChange: (page: number) => void;
}

const Pagination = ({pageIndex, pageCount, onPageChange}: PaginationProps) => {
    if (pageCount == 0) return null;

    const getPages = () => {
        const pages: (number | string)[] = [];

        if (pageCount <= 5) {
            for (let i = 0; i < pageCount; i++) pages.push(i);
            return pages;
        }

        pages.push(0);

        if (pageIndex > 2) pages.push("...");

        const start = Math.max(1, pageIndex - 1);
        const end = Math.min(pageCount - 2, pageIndex + 1);

        for (let i = start; i <= end; i++) {
            pages.push(i);
        }

        if (pageIndex < pageCount - 3) pages.push("...");

        pages.push(pageCount - 1);

        return pages;
    };

    const pages = getPages();

    return (
        <div className={styles.pagination}>
            <button
                onClick={() => onPageChange(pageIndex - 1)}
                disabled={pageIndex === 0}
            >
                {"<"}
            </button>

            {pages.map((p, i) =>
                p === "..." ? (
                    <span key={i}>...</span>
                ) : (
                    <button
                        key={i}
                        onClick={() => onPageChange(p as number)}
                        className={p === pageIndex ? styles.active : ""}
                    >
                        {Number(p) + 1}
                    </button>
                )
            )}

            <button
                onClick={() => onPageChange(pageIndex + 1)}
                disabled={pageIndex >= pageCount - 1}
            >
                {">"}
            </button>
        </div>
    );
};

export default Pagination;