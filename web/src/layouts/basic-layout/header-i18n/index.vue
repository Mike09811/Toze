<template>
    <Dropdown :trigger="isMobile ? 'click' : 'hover'" class="i-layout-header-i18n" :class="{ 'i-layout-header-user-mobile': isMobile }" @on-click="handleClick">
        <span class="i-layout-header-trigger i-layout-header-trigger-min" :class="{ 'i-layout-header-trigger-no-height': outside }">
            <Icon custom="i-icon i-icon-i18n" />
        </span>
        <template #list>
            <DropdownMenu>
                <DropdownItem v-for="item in languages" :key="item.locale" :name="item.locale" :selected="locale === item.locale">
                    <img v-if="item.icon" :src="item.icon">
                    <span>{{ item.language }}</span>
                </DropdownItem>
            </DropdownMenu>
        </template>
    </Dropdown>
</template>
<script>
    import Setting from '@/setting';
    import { mapState, mapActions } from 'vuex';

    export default {
        name: 'iHeaderI18n',
        props: {
            outside: {
                type: Boolean,
                default: false
            }
        },
        data () {
            return {
                languages: Setting.i18n.list
            }
        },
        computed: {
            ...mapState('admin/i18n', [
                'locale'
            ]),
            ...mapState('admin/layout', [
                'isMobile'
            ])
        },
        methods: {
            ...mapActions('admin/i18n', [
                'setLocale'
            ]),
            handleClick (locale) {
                if (locale === this.locale) return;
                this.setLocale({ locale, vm: this });
            }
        }
    }
</script>
