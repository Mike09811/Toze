import request from '@/plugins/request';

// ---------- 服务器管理 ----------

export function GetPaasServerList (data) {
    return request({
        url: '/api/paas/server/list',
        method: 'post',
        data
    });
}

export function CreatePaasServer (data) {
    return request({
        url: '/api/paas/server/create',
        method: 'post',
        data
    });
}

export function UpdatePaasServer (data) {
    return request({
        url: '/api/paas/server/update',
        method: 'post',
        data
    });
}

export function DeletePaasServer (data) {
    return request({
        url: '/api/paas/server/delete',
        method: 'post',
        data
    });
}

export function TestPaasServer (data) {
    return request({
        url: '/api/paas/server/test',
        method: 'post',
        data
    });
}

export function InitPaasServer (data) {
    return request({
        url: '/api/paas/server/init',
        method: 'post',
        data
    });
}

// ---------- 应用管理 ----------

export function GetPaasAppList (data) {
    return request({
        url: '/api/paas/app/list',
        method: 'post',
        data
    });
}

export function CreatePaasApp (data) {
    return request({
        url: '/api/paas/app/create',
        method: 'post',
        data
    });
}

export function UpdatePaasApp (data) {
    return request({
        url: '/api/paas/app/update',
        method: 'post',
        data
    });
}

export function DeletePaasApp (data) {
    return request({
        url: '/api/paas/app/delete',
        method: 'post',
        data
    });
}

export function DeployPaasApp (data) {
    return request({
        url: '/api/paas/app/deploy',
        method: 'post',
        data
    });
}

export function StartPaasApp (data) {
    return request({
        url: '/api/paas/app/start',
        method: 'post',
        data
    });
}

export function StopPaasApp (data) {
    return request({
        url: '/api/paas/app/stop',
        method: 'post',
        data
    });
}

export function RestartPaasApp (data) {
    return request({
        url: '/api/paas/app/restart',
        method: 'post',
        data
    });
}

export function GetPaasAppLogs (data) {
    return request({
        url: '/api/paas/app/logs',
        method: 'post',
        data
    });
}

export function RefreshPaasAppStatus (data) {
    return request({
        url: '/api/paas/app/status',
        method: 'post',
        data
    });
}

// ---------- 域名管理 ----------

export function GetPaasDomainList (data) {
    return request({
        url: '/api/paas/domain/list',
        method: 'post',
        data
    });
}

export function CreatePaasDomain (data) {
    return request({
        url: '/api/paas/domain/create',
        method: 'post',
        data
    });
}

export function DeletePaasDomain (data) {
    return request({
        url: '/api/paas/domain/delete',
        method: 'post',
        data
    });
}
