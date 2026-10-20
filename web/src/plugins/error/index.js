import { nextTick } from 'vue';
import store from '@/store';
import util from '@/libs/util';

export default {
    install (app, options) {
        app.config.errorHandler = function (error, instance, info) {
            nextTick(() => {
                // store 追加 log
                store.dispatch('admin/log/push', {
                    message: `${info}: ${error.message}`,
                    type: 'error',
                    meta: {
                        error
                        // instance
                    }
                });
                // 只在开发模式下打印 log
                if (process.env.NODE_ENV === 'development') {
                    util.log.capsule('Admin Plus', 'ErrorHandler', 'error');
                    util.log.error('>>>>>> 错误信息 >>>>>>');
                    console.log(info);
                    util.log.error('>>>>>> Vue 实例 >>>>>>');
                    console.log(instance);
                    util.log.error('>>>>>> Error >>>>>>');
                    console.log(error)
                }
            })
        }
    }
}
