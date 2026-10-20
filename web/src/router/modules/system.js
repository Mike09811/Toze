import BasicLayout from '@/layouts/basic-layout';

const meta = {
    auth: true
};

const pre = 'system-';

export default {
    path: '/system',
    name: 'system',
    redirect: {
        name: `${pre}user`
    },
    meta,
    component: BasicLayout,
    children: [
        {
            path: 'user',
            name: `${pre}user`,
            meta: {
                ...meta,
                title: '用户管理'
            },
            component: () => import('@/pages/sys/user')
        }
    ]
};
