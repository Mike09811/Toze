/**
 * 菜单
 * */
import { cloneDeep } from 'lodash';
import { includeArray, setNativeMenuSider, setNativeMenuHeader, getHeaderName, getMenuSider, getSiderSubmenu, getNativeMenuSider, getNativeMenuHeader } from '@/libs/system';
import router from '@/router';
import { MenuList } from '@api/menu';
import Setting from '@/setting';
import menuSider from '@/menu/sider';

// 根据 menu 配置的权限，过滤菜单
function filterMenu (menuList, access, lastList) {
    menuList.forEach(menu => {
        let menuAccess = menu.auth;

        if (!menuAccess || includeArray(menuAccess, access)) {
            let newMenu = {};
            for (let i in menu) {
                if (i !== 'children') newMenu[i] = cloneDeep(menu[i]);
            }
            if (menu.children && menu.children.length) newMenu.children = [];

            lastList.push(newMenu);
            menu.children && filterMenu(menu.children, access, newMenu.children);
        }
    });
    return lastList;
}

export default {
    namespaced: true,
    state: {
        // 顶部菜单
        header: [],
        // 侧栏菜单
        sider: [],
        // 当前顶栏菜单的 name
        headerName: '',
        // 当前所在菜单的 path
        activePath: '',
        // 展开的子菜单 name 集合
        openNames: [],
        // 所有的菜单
        menuSider: [],
        // 侧边菜单徽标
        siderMenuBadge: Setting.badge.siderMenuBadge,
        // 顶栏菜单徽标
        headerMenuBadge: Setting.badge.headerMenuBadge
    },
    getters: {
        /**
         * @description 根据 user 里登录用户权限，对侧边菜单进行鉴权过滤
         * */
        filterSider (state, getters, rootState) {
            const userInfo = rootState.admin.user.info;
            // @权限
            const access = userInfo.access;
            if (access && access.length) {
                return filterMenu(state.sider, access, []);
            } else {
                return filterMenu(state.sider, [], []);
            }
        },
        /**
         * @description 根据 user 里登录用户权限，对顶栏菜单进行鉴权过滤
         * */
        filterHeader (state, getters, rootState) {
            const userInfo = rootState.admin.user.info;
            // @权限
            const access = userInfo.access;
            if (access && access.length) {
                return state.header.filter(item => {
                    let state = true;
                    if (item.auth && !includeArray(item.auth, access)) state = false;
                    if (item.children && item.children.length) {
                        item.children = item.children.filter(child => {
                            let state = true;
                            if (child.auth && !includeArray(child.auth, access)) state = false;
                            return state;
                        })
                    }
                    return state;
                });
            } else {
                return state.header.filter(item => {
                    let state = true;
                    if (item.auth && item.auth.length) state = false;
                    if (item.children && item.children.length) {
                        item.children = item.children.filter(child => {
                            let state = true;
                            if (child.auth && child.auth.length) state = false;
                            return state;
                        });
                    }
                    return state;
                });
            }
        },
        /**
         * @description 当前 header 的全部信息
         * */
        currentHeader (state) {
            return state.header.find(item => item.name === state.headerName);
        },
        /**
         * @description 在当前 header 下，是否隐藏 sider（及折叠按钮）
         * */
        hideSider (state, getters) {
            let visible = false;
            if (getters.currentHeader && 'hideSider' in getters.currentHeader) visible = getters.currentHeader.hideSider;
            return visible;
        }
    },
    mutations: {
        /**
         * @description 设置侧边栏菜单
         * @param {Object} state vuex state
         * @param {Array} menu menu
         */
        setSider (state, menu) {
            state.sider = menu;
        },
        /**
         * @description 设置顶栏菜单
         * @param {Object} state vuex state
         * @param {Array} menu menu
         */
        setHeader (state, menu) {
            state.header = menu;
        },
        /**
         * @description 设置当前顶栏菜单 name
         * @param {Object} state vuex state
         * @param {Array} name headerName
         */
        setHeaderName (state, name) {
            state.headerName = name;
        },
        /**
         * @description 设置当前所在菜单的 path，用于侧栏菜单高亮当前项
         * @param {Object} state vuex state
         * @param {Array} path fullPath
         */
        setActivePath (state, path) {
            state.activePath = path;
        },
        /**
         * @description 设置当前所在菜单的全部展开父菜单的 names 集合
         * @param {Object} state vuex state
         * @param {Array} names openNames
         */
        setOpenNames (state, names) {
            state.openNames = names;
        },
        /**
         * @description 设置所有菜单
         * @param {Object} state vuex state
         * @param {Array} menuSider menuSider
         * */
        setMenuSider (state, menuSider) {
            state.menuSider = menuSider;
        },
        /**
         * @description 设置全部的侧边菜单的徽标
         * */
        setAllSiderMenuBadge (state, data) {
            state.siderMenuBadge = data;
        },
        /**
         * @description 新增或修改某个侧边菜单的徽标
         * */
        setSiderMenuBadge (state, { path, badge }) {
            const siderMenuBadge = cloneDeep(state.siderMenuBadge);
            const menuIndex = siderMenuBadge.findIndex(item => item.path === path);
            if (menuIndex >= 0) {
                siderMenuBadge[menuIndex] = badge;
                state.siderMenuBadge = siderMenuBadge;
            } else {
                state.siderMenuBadge.push(badge);
            }
        },
        /**
         * @description 删除某个侧边菜单的徽标
         * */
        removeSiderMenuBadge (state, path) {
            const menuIndex = state.siderMenuBadge.findIndex(item => item.path === path);
            if (menuIndex >= 0) state.siderMenuBadge.splice(menuIndex, 1);
        },
        /**
         * @description 设置全部的顶栏菜单的徽标
         * */
        setAllHeaderMenuBadge (state, data) {
            state.headerMenuBadge = data;
        },
        /**
         * @description 新增或修改某个顶栏菜单的徽标
         * */
        setHeaderMenuBadge (state, { path, badge }) {
            const headerMenuBadge = cloneDeep(state.headerMenuBadge);
            const menuIndex = headerMenuBadge.findIndex(item => item.path === path);
            if (menuIndex >= 0) {
                headerMenuBadge[menuIndex] = badge;
                state.headerMenuBadge = headerMenuBadge;
            } else {
                state.headerMenuBadge.push(badge);
            }
        },
        /**
         * @description 删除某个顶栏菜单的徽标
         * */
        removeHeaderMenuBadge (state, path) {
            const menuIndex = state.headerMenuBadge.findIndex(item => item.path === path);
            if (menuIndex >= 0) state.headerMenuBadge.splice(menuIndex, 1);
        }
    },
    actions: {
        /**
         * @description 设置菜单（动态+静态）
         * */
        setMenuList ({ dispatch, commit }, to = {}) {
            // 只动态菜单设置顶栏菜单
            if (Setting.dynamicMenu) {
                // 设置顶栏菜单
                const menuHeaderList = getNativeMenuHeader();
                commit('setHeader', menuHeaderList);
            }
            // 设置侧边栏菜单
            const menuSiderList = Setting.dynamicMenu ? getNativeMenuSider() : [...menuSider];
            let path = to.matched[to.matched.length - 1].path;

            let headerName = getHeaderName(path, menuSiderList);
            if (headerName === null) {
                path = to.path;
                headerName = getHeaderName(path, menuSiderList);
            }
            // 在 404 时，是没有 headerName 的
            if (headerName !== null) {
                commit('setHeaderName', headerName);
                commit('setMenuSider', menuSiderList);

                const filterMenuSider = getMenuSider(menuSiderList, headerName);
                commit('setSider', filterMenuSider);
                commit('setActivePath', to.path);

                const openNames = getSiderSubmenu(path, menuSiderList);
                commit('setOpenNames', openNames);
            }
        },
        /**
         * @description 动态获取菜单
         * @param dispatch
         * @param loadMenu 是否加载菜单，需传 $route 信息
         * */
        getMenuList ({ dispatch }, loadMenu) {
            return new Promise((resolve, reject) => {
                MenuList()
                    .then(res => {
                        setNativeMenuHeader(res.header);
                        setNativeMenuSider(res.sider);
                        if (loadMenu) {
                            const to = loadMenu === true ? router.currentRoute.value : loadMenu;
                            dispatch('setMenuList', to);
                        }
                        resolve();
                    })
                    .catch(err => {
                        reject(err);
                    });
            });
        }
    }
}
