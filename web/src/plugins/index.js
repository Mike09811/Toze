/**
 * 插件
 * */

// 错误捕获
import pluginError from '@/plugins/error';
// 日志插件
import pluginLog from '@/plugins/log';
// 鉴权指令
import directiveAuth from '@/plugins/auth';

export default {
    async install (app, options) {
        // 插件
        app.use(pluginError);
        app.use(pluginLog);

        // 指令
        app.directive('auth', directiveAuth);
    }
}
