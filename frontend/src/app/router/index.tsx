import {Navigate, Route, Routes} from 'react-router-dom';
import Layout from '@/widgets/Layout';
import ReportsPage from '@/pages/ReportsPage';
import NotFoundPage from '@/pages/NotFoundPage';
import ProfilePage from "@/pages/ProfilePage";
import TablePage from "@/pages/TablePage";

const Router = () => {
    return (
        <Routes>
            <Route path="/" element={<Layout/>}>
                <Route index element={<Navigate to="/profile" replace/>}/>

                <Route path="profile" element={<ProfilePage/>}/>
                <Route path="reports" element={<ReportsPage/>}/>
                <Route path=":type/:docName" element={<TablePage/>}/>

                <Route path="*" element={<NotFoundPage/>}/>
            </Route>
        </Routes>
    );
};

export default Router;