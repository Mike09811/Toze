import cookies from './util.cookies';
import log from './util.log';
import db from './util.db';

import Setting from '@/setting';

const util = {
    cookies,
    log,
    db
};

function tTitle (title = '') {
    const $t = window ? window.$app.$t : null;
    if ($t) {
        if (title.indexOf('$t:') === 0) {
            return $t(title.split('$t:')[1]);
        } else {
            return title;
        }
    } else {
        return title;
    }
}

/**
 * @description 更改标题
 * @param {Object} title 标题
 * @param {Object} count 未读消息数提示（可视情况选择使用或不使用）
 */
util.title = function ({ title, count }) {
    title = tTitle(title);
    let fullTitle;

    if (Setting.titleSuffix === false || Setting.titleSuffix === '') {
        fullTitle = title ? `${title}` : '';
    } else {
        fullTitle = title ? `${title} - ${Setting.titleSuffix}` : Setting.titleSuffix;
    }

    if (count) fullTitle = `(${count}条消息)${fullTitle}`;
    window.document.title = fullTitle;
};

function requestAnimation (task) {
    if ('requestAnimationFrame' in window) {
        return window.requestAnimationFrame(task);
    }

    setTimeout(task, 16);
}

export { requestAnimation };

export default util;
