import request from '@/plugins/request';

export function MenuList (data) {
    return request({
        url: '/api/system/menu/list',
        method: 'post',
        data
    });
}
