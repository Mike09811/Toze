import Cookies from 'js-cookie';
import Setting from '@/setting';

const cookies = {};

/**
 * @description 存储 cookie 值
 * @param {String} name cookie name
 * @param {String} value cookie value
 * @param {Object} cookieSetting cookie setting
 */
cookies.set = function (name = 'default', value = '', cookieSetting = {}) {
    let currentCookieSetting = {};

    if (Setting.cookiesExpires !== 0) currentCookieSetting.expires = Setting.cookiesExpires;

    Object.assign(currentCookieSetting, cookieSetting);
    Cookies.set(`admin-plus-${Setting.appID}-${name}`, value, currentCookieSetting);
};

/**
 * @description 拿到 cookie 值
 * @param {String} name cookie name
 */
cookies.get = function (name = 'default') {
    return Cookies.get(`admin-plus-${Setting.appID}-${name}`);
};

/**
 * @description 拿到 cookie 全部的值
 */
cookies.getAll = function () {
    return Cookies.get();
};

/**
 * @description 删除 cookie
 * @param {String} name cookie name
 */
cookies.remove = function (name = 'default') {
    return Cookies.remove(`admin-plus-${Setting.appID}-${name}`);
};

export default cookies;
