import BasicLayout from '@/layouts/basic-layout';

const meta = {
    auth: true
};

const pre = 'paas-';

export default {
    path: '/paas',
    name: 'paas',
    redirect: {
        name: `${pre}server`
    },
    meta,
    component: BasicLayout,
    children: [
        {
            path: 'server',
            name: `${pre}server`,
            meta: {
                ...meta,
                title: '服务器管理'
            },
            component: () => import('@/pages/paas/server')
        },
        {
            path: 'app',
            name: `${pre}app`,
            meta: {
                ...meta,
                title: '应用管理'
            },
            component: () => import('@/pages/paas/app')
        },
        {
            path: 'domain',
            name: `${pre}domain`,
            meta: {
                ...meta,
                title: '域名管理'
            },
            component: () => import('@/pages/paas/domain')
        }
    ]
};
