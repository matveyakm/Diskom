import {useEffect, useRef, useState} from "react";
import styles from "./NavPanel.module.scss";
import {Link} from "react-router-dom";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import CloseIcon from '@/assets/icons/arrow_down.svg?react';

interface Props {
    activeRoot: MenuItem | null;
    allItems: MenuItem[];
    closePanel: () => void;
}

const NavPanel = ({activeRoot, allItems, closePanel}: Props) => {
    const [safeRoot, setSafeRoot] = useState<MenuItem | null>(null);
    const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

    useEffect(() => {
        if (activeRoot) {
            clearTimeout(timerRef.current);
            setSafeRoot(activeRoot);
        } else {
            timerRef.current = setTimeout(() => {
                setSafeRoot(null);
            }, 200);
        }
        return () => clearTimeout(timerRef.current);
    }, [activeRoot]);

    const isOpen = activeRoot !== null;

    const renderTree = (parentId: number, currentLevel: number) => {
        const currentLevelItems = allItems.filter(
            item => item.id_group === parentId && item.level === currentLevel
        );

        if (currentLevelItems.length === 0) return null;

        return currentLevelItems.map(item => {
            const hasChildren = allItems.some(i => i.id_group === item.id && i.level === currentLevel + 1);

            return (
                <li key={item.id} className={styles.menuItem}>
                    {hasChildren || !item.ref || item.ref === "#" ? (
                        <div className={styles.menuItemContent}>
                            <span>{item.name}</span>
                        </div>
                    ) : (
                        <Link
                            to={`/table/${item.ref}`}
                            className={styles.menuItemContent}
                            onClick={closePanel}
                        >
                            <span>{item.name}</span>
                        </Link>
                    )}

                    {hasChildren && (
                        <ul className={styles.submenu}>
                            {renderTree(item.id, currentLevel + 1)}
                        </ul>
                    )}
                </li>
            );
        });
    };

    return (
        <div className={`${styles.panel} ${isOpen ? styles.open : ''}`}>
            {safeRoot && (
                <>
                    <div className={styles.header}>
                        <span className={styles.title}>{safeRoot.name}</span>
                        <div className={styles.close} onClick={closePanel}><CloseIcon/></div>
                    </div>

                    <ul className={styles.menuList}>
                        {renderTree(safeRoot.id, safeRoot.level + 1)}
                    </ul>
                </>
            )}
        </div>
    );
};

export default NavPanel;