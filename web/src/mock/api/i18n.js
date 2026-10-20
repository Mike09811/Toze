export default [
    {
        path: '/api/system/i18n/list',
        method: 'post',
        handle ({ body }) {
            return {
                code: 0,
                msg: '',
                data: {
                    'zh-CN': {
                        menu: {
                            dashboard: '主控台'
                        }
                    },
                    'en-US': {
                        menu: {
                            dashboard: 'Dashboard'
                        }
                    }
                }
            }
        }
    }
]