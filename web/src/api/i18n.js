import request from '@/plugins/request';

export function I18nList (data) {
    return request({
        url: '/api/system/i18n/list',
        method: 'post',
        data
    });
}
