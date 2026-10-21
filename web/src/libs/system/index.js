/**
 * 系统内置方法集，正常情况下您不应该修改或移除此文件
 * */

import { cloneDeep } from 'lodash';
import Setting from '@/setting';

/**
 * @description 根据当前路由，找打顶部菜单名称
 * @param {String} currentPath 当前路径
 * @param {Array} menuList 所有路径
 * */
function getHeaderName (currentPath, menuList) {
    const allMenus = [];
    menuList.forEach(menu => {
        const headerName = menu.header || '';
        const menus = transferMenu(menu, headerName);
        allMenus.push({
            path: menu.path,
            header: headerName
        });
        menus.forEach(item => allMenus.push(item));
    });

    let currentMenu = allMenus.find(item => item.path === currentPath);
    // 前缀匹配：直接刷新/打开隐藏的二级页面（如 /paas/app/detail/2）时，
    // 也要能定位到所属顶栏菜单，否则侧边栏会空白
    if (!currentMenu) {
        const candidates = allMenus.filter(item => item.path !== '/' && currentPath.startsWith(item.path + '/'));
        candidates.sort((a, b) => b.path.length - a.path.length);
        currentMenu = candidates[0];
    }
    // 兜底：未匹配到时回到首页菜单，保证侧边栏可用
    if (!currentMenu) {
        currentMenu = allMenus.find(item => item.path === '/');
    }
    return currentMenu ? currentMenu.header : null;
}

function transferMenu (menu, headerName) {
    if (menu.children && menu.children.length) {
        return menu.children.reduce((all, item) => {
            all.push({
                path: item.path,
                header: headerName
            });
            const foundChildren = transferMenu(item, headerName);
            return all.concat(foundChildren);
        }, []);
    } else {
        return [menu];
    }
}

export { getHeaderName };

/**
 * @description 根据当前顶栏菜单 name，找到对应的二级菜单
 * @param {Array} menuList 所有的二级菜单
 * @param {String} headerName 当前顶栏菜单的 name
 * */
function getMenuSider (menuList, headerName = '') {
    if (headerName) {
        return menuList.filter(item => item.header === headerName);
    } else {
        return menuList;
    }
}

export { getMenuSider };

/**
 * @description 根据当前路由找到对应的菜单 path（支持前缀匹配），
 * 用于侧栏高亮：/paas/app/detail/2 高亮「应用管理」
 * */
function getMatchedPath (currentPath, menuList) {
    const allPaths = [];
    menuList.forEach(menu => {
        allPaths.push(menu.path);
        transferMenu(menu, menu.header || '').forEach(item => allPaths.push(item.path));
    });
    if (allPaths.includes(currentPath)) return currentPath;
    const candidates = allPaths.filter(p => p !== '/' && currentPath.startsWith(p + '/'));
    candidates.sort((a, b) => b.length - a.length);
    return candidates[0] || currentPath;
}

export { getMatchedPath };

/**
 * @description 根据当前路由，找到其所有父菜单 path，作为展开侧边栏 open-names 依据
 * @param {String} currentPath 当前路径
 * @param {Array} menuList 所有路径
 * */
function getSiderSubmenu (currentPath, menuList) {
    const allMenus = [];
    menuList.forEach(menu => {
        const menus = transferSubMenu(menu, []);
        allMenus.push({
            path: menu.path,
            openNames: []
        });
        menus.forEach(item => allMenus.push(item));
    });

    const currentMenu = allMenus.find(item => item.path === currentPath);
    return currentMenu ? currentMenu.openNames : [];
}

function transferSubMenu (menu, openNames) {
    if (menu.children && menu.children.length) {
        const itemOpenNames = openNames.concat([menu.path]);
        return menu.children.reduce((all, item) => {
            all.push({
                path: item.path,
                openNames: itemOpenNames
            });
            const foundChildren = transferSubMenu(item, itemOpenNames);
            return all.concat(foundChildren);
        }, []);
    } else {
        return [menu].map(item => {
            return {
                path: item.path,
                openNames: openNames
            }
        });
    }
}

export { getSiderSubmenu };

/**
 * @description 递归获取所有子菜单
 * */
function getAllSiderMenu (menuList) {
    let allMenus = [];

    menuList.forEach(menu => {
        if (menu.children && menu.children.length) {
            const menus = getMenuChildren(menu);
            menus.forEach(item => allMenus.push(item));
        } else {
            allMenus.push(menu);
        }
    });

    return allMenus;
}

function getMenuChildren (menu) {
    if (menu.children && menu.children.length) {
        return menu.children.reduce((all, item) => {
            const foundChildren = getMenuChildren(item);
            return all.concat(foundChildren);
        }, []);
    } else {
        return [menu];
    }
}

export { getAllSiderMenu };

/**
 * @description 将菜单转为平级
 * */
function flattenSiderMenu (menuList, newList) {
    menuList.forEach(menu => {
        let newMenu = {};
        for (let i in menu) {
            if (i !== 'children') newMenu[i] = cloneDeep(menu[i]);
        }
        newList.push(newMenu);
        menu.children && flattenSiderMenu(menu.children, newList);
    });
    return newList;
}

export { flattenSiderMenu };

/**
 * @description 判断列表1中是否包含了列表2中的某一项
 * 因为用户权限 access 为数组，includes 方法无法直接得出结论
 * */
function includeArray (list1, list2) {
    let status = false;
    list2.forEach(item => {
        if (list1.includes(item)) status = true;
    });
    return status;
}

export { includeArray };

/**
 * @description 动态菜单相关
 * */

// 部分内嵌浏览器/隐私模式下 localStorage 可能不可用或抛异常，
// 这里统一做保护并退回到内存缓存，保证侧边栏永远不会因为存储问题而空白。
const menuStorageCache = {
    sider: '[]',
    header: '[]'
};

function safeStorageSet (key, value) {
    menuStorageCache[key] = value;
    try {
        localStorage.setItem(`admin-plus-${Setting.appID}-menu-${key}`, value);
    } catch (e) {}
}

function safeStorageGet (key) {
    try {
        const v = localStorage.getItem(`admin-plus-${Setting.appID}-menu-${key}`);
        if (v) menuStorageCache[key] = v;
    } catch (e) {}
    try {
        return JSON.parse(menuStorageCache[key] || '[]');
    } catch (e) {
        return [];
    }
}

/**
 * @description 保存动态侧边菜单到本地
 * */
function setNativeMenuSider (data) {
    safeStorageSet('sider', JSON.stringify(data || []));
}

export { setNativeMenuSider };

/**
 * @description 读取本地的动态侧边菜单
 * */
function getNativeMenuSider () {
    return safeStorageGet('sider');
}

export { getNativeMenuSider };

/**
 * @description 保存动态顶栏菜单到本地
 * */
function setNativeMenuHeader (data) {
    safeStorageSet('header', JSON.stringify(data || []));
}

export { setNativeMenuHeader };

/**
 * @description 读取本地的动态顶栏菜单
 * */
function getNativeMenuHeader () {
    return safeStorageGet('header');
}

export { getNativeMenuHeader };
