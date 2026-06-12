import styles from './Layout.module.scss';
import {Outlet} from 'react-router-dom';
import Sidebar from '@/widgets/Sidebar';

const Layout = () => {
    return (
        <div className={styles.layout}>
            <Sidebar/>
            <div className={styles.content}>
                <main className={styles.main}>
                    <Outlet/>
                </main>
            </div>
        </div>
    );
};

export default Layout;