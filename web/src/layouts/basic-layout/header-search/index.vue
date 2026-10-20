<template>
    <span v-if="isDesktop" class="i-layout-header-trigger i-layout-header-trigger-min i-layout-header-trigger-in i-layout-header-trigger-nohover">
        <input class="i-layout-header-search" type="text" :placeholder="$t('basicLayout.search.placeholder')">
    </span>
    <Dropdown v-else trigger="click" class="i-layout-header-search-drop" ref="dropdown">
        <span class="i-layout-header-trigger i-layout-header-trigger-min">
            <Icon type="ios-search" />
        </span>
        <template #list>
            <DropdownMenu>
                <Row align="middle">
                    <Col flex="auto" class="ivu-pl-4">
                        <Input size="large" prefix="ios-search" :placeholder="$t('basicLayout.search.placeholder')" />
                    </Col>
                    <Col flex="80px" class="ivu-text-center">
                        <span @click="handleCloseSearch">{{ $t('basicLayout.search.cancel') }}</span>
                    </Col>
                </Row>
            </DropdownMenu>
        </template>
    </Dropdown>
</template>
<script>
    import { mapState } from 'vuex';

    export default {
        name: 'iHeaderSearch',
        computed: {
            ...mapState('admin/layout', [
                'isDesktop',
                'headerMenu'
            ])
        },
        methods: {
            handleCloseSearch () {
                this.$refs.dropdown.handleClick();
            }
        }
    }
</script>
