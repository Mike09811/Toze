/**
 * 自动配置各类语言包
 * */
import { merge } from 'lodash';
import Setting from '@/setting';
import { I18nList } from '@api/i18n';

// 导入 View UI Plus 语言包
const ViewUIPlusLocale = require.context('view-ui-plus/dist/locale', false, /.+\-.+.js$/);

// 导入布局语言包
import layoutLocale from '@/layouts/basic-layout/i18n';

// 页面语言包
const pageLocale = require.context('@/pages', true, /i18n.js/);

// 菜单语言包
const menuLocale = require.context('@/menu', true, /i18n.js/);

// 组件语言包
const componentLocale = require.context('@/components', true, /i18n.js/);

/**
 * @description 配置本地语言包
 * */
function loadNativeLocaleMessages () {
    const messages = {};

    for (const lang of Setting.i18n.list) {
        messages[lang.locale] = {
            page: {},
            menu: {},
            component: {}
        };

        // 设置 View UI Plus 语言包
        ViewUIPlusLocale.keys().forEach(path => {
            const currentLocale = ViewUIPlusLocale(path).default;
            if (currentLocale.i.locale === lang.locale) {
                Object.keys(currentLocale).forEach(key => {
                    messages[lang.locale][key] = currentLocale[key];
                });
            }
        });

        // 设置 Layout 语言包
        Object.keys(layoutLocale).forEach(locale => {
            if (lang.locale === locale) {
                Object.keys(layoutLocale[locale]).forEach(key => {
                    messages[lang.locale][key] = layoutLocale[locale][key];
                });
            }
        });

        // 设置页面语言包
        pageLocale.keys().forEach(path => {
            const currentPage = pageLocale(path).default[lang.locale];
            if (currentPage) {
                Object.keys(currentPage).forEach(key => {
                    messages[lang.locale].page[key] = currentPage[key];
                });
            }
        });

        // 设置菜单语言包
        menuLocale.keys().forEach(path => {
            const currentMenu = menuLocale(path).default[lang.locale];
            if (currentMenu) {
                Object.keys(currentMenu).forEach(key => {
                    messages[lang.locale].menu[key] = currentMenu[key];
                });
            }
        });

        // 设置组件语言包
        componentLocale.keys().forEach(path => {
            const currentComponent = componentLocale(path).default[lang.locale];
            if (currentComponent) {
                Object.keys(currentComponent).forEach(key => {
                    messages[lang.locale].component[key] = currentComponent[key];
                });
            }
        });
    }
    return messages;
}

export default loadNativeLocaleMessages();

/**
 * @description 配置远程接口语言包
 * */
export async function loadRemoteLocaleMessages (i18n) {
    const nativeMessages = loadNativeLocaleMessages();
    const remoteMessages = await I18nList();
    const messages = merge({}, nativeMessages, remoteMessages);
    Object.keys(messages).forEach(key => {
        i18n.global.setLocaleMessage(key, messages[key]);
    });
}
