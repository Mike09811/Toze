import headerMenu from '@/menu/header';
import siderMenu from '@/menu/sider';

export default [
    {
        path: '/api/system/menu/list',
        method: 'post',
        handle ({ body }) {
            return {
                code: 0,
                msg: '',
                data: {
                    header: headerMenu,
                    sider: siderMenu
                }
            }
        }
    }
]