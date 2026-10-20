<template>
    <div id="app">
        <router-view />
    </div>
</template>
<script>
    import { mapMutations } from 'vuex';

    import { on, off } from 'view-ui-plus/src/utils/dom';
    import { setMatchMedia } from 'view-ui-plus/src/utils/assist';

    setMatchMedia();

    export default {
        name: 'app',
        methods: {
            ...mapMutations('admin/layout', [
                'setDevice',
                'setBodyHeight'
            ]),
            handleWindowResize () {
                this.handleMatchMedia();
                this.handleSetBodyHeight();
            },
            handleMatchMedia () {
                const matchMedia = window.matchMedia;

                if (matchMedia('(max-width: 600px)').matches) {
                    this.setDevice('Mobile');
                } else if (matchMedia('(max-width: 992px)').matches) {
                    this.setDevice('Tablet');
                } else {
                    this.setDevice('Desktop');
                }
            },
            handleSetBodyHeight () {
                this.setBodyHeight(document.body.offsetHeight);
            }
        },
        mounted () {
            on(window, 'resize', this.handleWindowResize);
            this.handleMatchMedia();
            this.handleSetBodyHeight();
        },
        beforeUnmount () {
            off(window, 'resize', this.handleWindowResize);
        }
    }
</script>
