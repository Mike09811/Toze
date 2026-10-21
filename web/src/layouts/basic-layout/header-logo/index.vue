<template>
    <i-link class="i-layout-header-logo toze-logo" :class="{ 'i-layout-header-logo-stick': !isMobile }" to="/">
        <Icon type="md-rocket" class="toze-logo-icon" />
        <span class="toze-logo-name" v-if="!isMobile">Toze</span>
    </i-link>
</template>
<script>
    import { mapState } from 'vuex';

    export default {
        name: 'iHeaderLogo',
        computed: {
            ...mapState('admin/layout', [
                'isMobile',
                'headerTheme'
            ])
        }
    }
</script>
