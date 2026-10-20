<template>
    <div>
        <MenuItem v-bind="$attrs" :to="menu.path" :replace="menu.replace" :target="menu.target" :name="menu.path" @click="handleClick(menu.path)">
            <i-menu-side-title :menu="menu" :hide-title="hideTitle" />
            <Badge class="i-layout-menu-side-badge" v-if="badgeData" v-bind="badgeData" />
        </MenuItem>
    </div>
</template>
<script>
    import iMenuSideTitle from './menu-title';
    import clickItem from '../mixins/click-item';
    import siderMenuBadge from '../mixins/sider-menu-badge';

    export default {
        name: 'iMenuSideItem',
        components: { iMenuSideTitle },
        mixins: [ clickItem, siderMenuBadge ],
        props: {
            menu: {
                type: Object,
                default () {
                    return {}
                }
            },
            hideTitle: {
                type: Boolean,
                default: false
            }
        }
    }
</script>
