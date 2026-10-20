<template>
    <div class="i-layout-tabs" :class="classes" :style="styles">
        <div class="i-layout-tabs-main">
            <Tabs
                    type="card"
                    :model-value="current"
                    :animated="false"
                    closable
                    :draggable="tabsOrder"
                    @on-click="handleClickTab"
                    @on-tab-remove="handleClickClose"
                    @on-drag-drop="handleDragDrop"
            >
                <TabPane
                        v-for="(page, index) in opened"
                        :key="index"
                        :label="(h) => tabLabel(h, page)"
                        :name="page.fullPath"
                        :closable="page.meta && page.meta.closable"
                />
            </Tabs>
            <Dropdown class="i-layout-tabs-close" @on-click="handleClose">
                <div class="i-layout-tabs-close-main">
                    <Icon type="ios-arrow-down" />
                </div>
                <template #list>
                    <DropdownMenu>
                        <DropdownItem name="left">
                            <Icon type="md-arrow-back" />
                            {{ $t('basicLayout.tabs.left') }}
                        </DropdownItem>
                        <DropdownItem name="right">
                            <Icon type="md-arrow-forward" />
                            {{ $t('basicLayout.tabs.right') }}
                        </DropdownItem>
                        <DropdownItem name="other">
                            <Icon type="md-close" />
                            {{ $t('basicLayout.tabs.other') }}
                        </DropdownItem>
                        <DropdownItem name="all">
                            <Icon type="md-close-circle" />
                            {{ $t('basicLayout.tabs.all') }}
                        </DropdownItem>
                    </DropdownMenu>
                </template>
            </Dropdown>
        </div>
    </div>
</template>
<script>
    import { resolveComponent } from 'vue';
    import { mapState, mapGetters, mapActions } from 'vuex';
    import tTitle from '../mixins/translate-title';

    import Setting from '@/setting';

    import { getAllSiderMenu } from '@/libs/system';

    import { cloneDeep } from 'lodash';
    import random from '@/libs/random_str';

    export default {
        name: 'iTabs',
        mixins: [ tTitle ],
        emits: ['on-reload'],
        computed: {
            ...mapState('admin/page', [
                'opened',
                'current'
            ]),
            ...mapState('admin/menu', [
                'menuSider'
            ]),
            ...mapState('admin/layout', [
                'showTabsIcon',
                'tabsFix',
                'tabsReload',
                'tabsOrder',
                'headerFix',
                'headerStick',
                'isMobile',
                'menuCollapse'
            ]),
            ...mapGetters('admin/menu', [
                'hideSider'
            ]),
            classes () {
                return {
                    'i-layout-tabs-fix': this.tabsFix
                }
            },
            isHeaderStick () {
                return this.hideSider;
            },
            allSiderMenu () {
                return getAllSiderMenu(this.menuSider);
            },
            styles () {
                let style = {};
                if (this.tabsFix && !this.headerFix) {
                    style.top = `${Setting.headerHeight - this.scrollTop}px`;
                }

                const menuWidth = this.isHeaderStick ? 0 : this.menuCollapse ? Setting.menuSideCollapseWidth : Setting.menuSideWidth;
                if (!this.isMobile && this.tabsFix) {
                    style.width = `calc(100% - ${menuWidth}px)`;
                    style.left = `${menuWidth}px`;
                }

                return style;
            }
        },
        data () {
            return {
                scrollTop: 0
            }
        },
        methods: {
            ...mapActions('admin/page', [
                'close',
                'closeLeft',
                'closeRight',
                'closeOther',
                'closeAll',
                'updateOpened'
            ]),
            tabLabel (h, page) {
                const title = h('span', this.tTitle(page.meta ? page.meta.title : '未命名') || '未命名');
                let slot = [];

                if (this.showTabsIcon) {
                    const fullPathWithoutQuery = page.fullPath.indexOf('?') >= 0 ? page.fullPath.split('?')[0] : page.fullPath;
                    const currentMenu = this.allSiderMenu.find(menu => menu.path === fullPathWithoutQuery) || {};

                    let icon;
                    if (currentMenu.icon) {
                        icon = h(resolveComponent('Icon'), {
                            type: currentMenu.icon
                        });
                    } else if (currentMenu.custom) {
                        icon = h(resolveComponent('Icon'), {
                            custom: currentMenu.custom
                        });
                    } else if (currentMenu.img) {
                        icon = h('img', {
                            src: currentMenu.img
                        });
                    }

                    if (icon) slot.push(icon);
                    slot.push(title);
                } else {
                    slot.push(title);
                }

                return h('div', {
                    class: 'i-layout-tabs-title'
                }, slot);
            },
            handleClickTab (tabName) {
                if (tabName === this.current) {
                    if (this.tabsReload) {
                        this.$emit('on-reload');
                    }
                } else {
                    const page = this.opened.find(page => page.fullPath === tabName);
                    const { name, params, query } = page;

                    if (page) this.$router.push({ name, params, query }, () => {});
                }
            },
            handleClickClose (tagName) {
                this.close({
                    tagName
                });
            },
            handleScroll () {
                if (this.tabsFix && !this.headerFix) {
                    const scrollTop = document.body.scrollTop + document.documentElement.scrollTop;
                    this.scrollTop = scrollTop > Setting.headerHeight ? Setting.headerHeight : scrollTop;
                }
            },
            handleClose (name) {
                const params = {
                    pageSelect: this.current
                };
                switch (name) {
                case 'left':
                    this.closeLeft(params);
                    break;
                case 'right':
                    this.closeRight(params);
                    break;
                case 'other':
                    this.closeOther(params);
                    break;
                case 'all':
                    this.closeAll();
                    break;
                }
            },
            handleDragDrop (name, newName, a, b) {
                let opened = cloneDeep(this.opened);
                opened.splice(b, 1, ...opened.splice(a, 1, opened[b]));
                this.updateOpened({ opened });
            },
            random () {
                return random(6);
            }
        },
        mounted () {
            document.addEventListener('scroll', this.handleScroll, { passive: true });
            this.handleScroll();
        },
        beforeUnmount () {
            document.removeEventListener('scroll', this.handleScroll);
        }
    }
</script>
