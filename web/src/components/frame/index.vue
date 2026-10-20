<template>
    <div class="i-frame">
        <iframe v-if="ready" :src="src" frameborder="0" class="i-frame-iframe"></iframe>
    </div>
</template>
<script>
    import { nextTick } from 'vue';

    const noMarginClassName = 'i-layout-content-main-no-margin';
    const noCopyrightClassName = 'i-copyright-hidden';
    const contextFlexClassName = 'i-layout-content-flex';
    const contextMainFlexClassName = 'i-layout-content-main-flex';

    export default {
        name: 'i-frame',
        props: {
            src: {
                type: String,
                default: ''
            },
            // 是否保留默认左右边距（24px）
            margin: {
                type: Boolean,
                default: false
            },
            // 是否显示底部版权
            copyright: {
                type: Boolean,
                default: false
            }
        },
        data () {
            return {
                ready: false,
                contentEl: null,
                copyrightEl: null,
                layoutContentEl: null,
                layoutContentMainEl: null
            }
        },
        watch: {
            margin: {
                handler (val) {
                    nextTick(() => {
                        if (this.contentEl) {
                            if (!val) {
                                this.contentEl.classList.add(noMarginClassName);
                            } else {
                                this.contentEl.classList.remove(noMarginClassName);
                            }
                        }
                    });
                },
                immediate: true
            }
        },
        mounted () {
            this.contentEl = document.getElementsByClassName('i-layout-content-main')[0];

            this.copyrightEl = document.getElementsByClassName('i-copyright')[0];
            if (this.copyrightEl && !this.copyright) this.copyrightEl.classList.add(noCopyrightClassName);

            this.layoutContentEl = document.getElementsByClassName('i-layout-content')[0];
            if (this.layoutContentEl) this.layoutContentEl.classList.add(contextFlexClassName);

            this.layoutContentMainEl = document.getElementsByClassName('i-layout-content-main')[0];
            if (this.layoutContentMainEl) this.layoutContentMainEl.classList.add(contextMainFlexClassName);

            nextTick(() => {
                this.ready = true;
            });
        },
        beforeUnmount () {
            if (this.contentEl) this.contentEl.classList.remove(noMarginClassName);
            if (this.copyrightEl) this.copyrightEl.classList.remove(noCopyrightClassName);
            if (this.layoutContentEl) this.layoutContentEl.classList.remove(contextFlexClassName);
            if (this.layoutContentMainEl) this.layoutContentMainEl.classList.remove(contextMainFlexClassName);
            this.contentEl = null;
            this.copyrightEl = null;
            this.layoutContentEl = null;
            this.layoutContentMainEl = null;
        }
    }
</script>
<style lang="less">
    .i-frame{
        position: absolute;
        top: 0;
        bottom: 0;
        left: 0;
        right: 0;
        overflow: hidden;
        overflow-y: auto;

        &-iframe{
            position: absolute;
            width: 100%;
            height: 100%;
            top: 0;
            bottom: 0;
            left: 0;
            right: 0;
        }
    }
</style>
