import React, {useRef, useState} from "react";
import styles from "./Sidebar.module.scss";
import Profile from "@/assets/icons/account_circle.svg?react";
import Panel from "./NavPanel";
import {Link} from "react-router-dom";
import {useOnClickOutside} from "usehooks-ts";
import DropdownMenu from "@/widgets/Sidebar/DropdownMenu";
import useMenuData from "@/shared/api/fetchMenuData";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import {DynamicIcon} from "@/shared/utils/dynamicIcon";
import { asElementRef, asElementRefs } from "@/shared/utils/refUtils";

const Sidebar = () => {
    const [activeRootItem, setActiveRootItem] = useState<MenuItem | null>(null);
    const [isMenuOpen, setIsMenuOpen] = useState<boolean>(false);

    const {data: menuItems, isLoading} = useMenuData();

    const sidebarRef = useRef<HTMLDivElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const menuRef = useRef<HTMLDivElement>(null);

    const closePanel = () => setActiveRootItem(null);

    useOnClickOutside(asElementRefs([sidebarRef, panelRef]), closePanel);
    useOnClickOutside(asElementRef(menuRef), () => setIsMenuOpen(false));

    const handleRootClick = (item: MenuItem) => {
        setActiveRootItem(prev => (prev?.id === item.id ? null : item));
    };

    // Profile caps (without backup yet)
    const [job] = useState('Председатель');
    const [name] = useState('Фальков Степан');
    const [email] = useState('st112382@student.spbu.ru');

    const rootButtons = menuItems?.filter(item => item.level === 0 && item.id < 4) ?? [];

    return (
        <>
            <nav className={styles.sidebar} ref={sidebarRef}>
                <div className={styles.upper}>
                    <div className={styles.header}>
                        <img src="/diskom_logo.png" className={styles.logo} alt="Logo"/>
                    </div>

                    <div className={styles.nav}>
                        {isLoading ? (<div className={styles.loader}>...</div>) : (
                            rootButtons.map(item => {

                                if (item.is_group) {
                                    const isActive = activeRootItem?.id === item.id;
                                    return (
                                        <button
                                            key={item.id}
                                            onClick={() => handleRootClick(item)}
                                            className={isActive ? styles.active : ""}
                                        >
                                            <DynamicIcon iconFromBackend={item.icon} className={styles.navIcon}/>
                                        </button>
                                    );
                                }

                                return (
                                    <Link key={item.id} to={item.ref} onClick={closePanel}>
                                        <button>
                                            <DynamicIcon iconFromBackend={item.icon} className={styles.navIcon}/>
                                        </button>
                                    </Link>
                                );
                            })
                        )}
                    </div>
                </div>

                <div className={styles.menu} ref={menuRef}>
                    <div
                        className={`${styles.user} ${isMenuOpen ? styles.open : ''}`}
                        onClick={() => {
                            setIsMenuOpen(prev => !prev);
                            closePanel();
                        }}
                    >
                        <Profile/>
                    </div>

                    <DropdownMenu
                        email={email}
                        job={job}
                        name={name}
                        isOpen={isMenuOpen}
                        toggleMenu={() => setIsMenuOpen(prev => !prev)}
                    />
                </div>
            </nav>

            <div ref={panelRef}>
                <Panel activeRoot={activeRootItem} closePanel={closePanel} allItems={menuItems ?? []}/>
            </div>
        </>
    );
};

export default Sidebar;