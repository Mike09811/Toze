<template>
    <GlobalFooter class="i-copyright" :links="links" :copyright="copyright" />
</template>
<script>
    export default {
        name: 'i-copyright',
        data () {
            return {
                links: [
                    {
                        title: 'GitHub',
                        key: 'GitHub',
                        href: 'https://github.com/Mike09811/Toze',
                        blankTarget: true
                    }
                ],
                copyright: 'Copyright © 2026 Toze'
            }
        }
    }
</script>
<style lang="less">
    .i-copyright{
        flex: 0 0 auto;
        &-hidden{
            display: none;
        }
    }
</style>
