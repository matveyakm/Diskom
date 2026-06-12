import styles from './DropdownMenu.module.scss';
import Profile from '@/assets/icons/account_circle.svg?react';
import Logout from '@/assets/icons/logout.svg?react';
import {Link} from "react-router-dom";
import {useThemeContext} from "@/app/providers/ThemeProvider";
import ThemeIcon from '@/assets/icons/theme.svg?react';

type Props = {
    email: string;
    job: string;
    name: string;
    isOpen: boolean;
    toggleMenu: () => void;
};

const DropdownMenu = ({email, job, name, isOpen, toggleMenu}: Props) => {

    const {toggleTheme} = useThemeContext();

    return (
        <div className={`${styles.dropdownMenu} ${isOpen ? styles.open : ''}`}>
            <div className={styles.infoUsers}>
                <div className={styles.infoName}>{name}</div>
                <div className={styles.infoMeta}>{job}</div>
                <div className={styles.infoMeta}>{email}</div>
            </div>
            <Link to="/profile" className={styles.dropdownLink} onClick={toggleMenu}>
                <Profile/>
                Профиль
            </Link>

            <div className={styles.dropdownLink} onClick={toggleTheme}>
                <ThemeIcon/>
                Тема
            </div>

            <div className={`${styles.dropdownLink} ${styles.dropdownLinkDanger}`}>
                <Logout/>
                Выйти
            </div>

        </div>
    );
};

export default DropdownMenu;