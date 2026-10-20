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
                        title: '官网',
                        key: '官网',
                        href: 'https://www.iviewui.com',
                        blankTarget: true
                    },
                    {
                        title: '社区',
                        key: '社区',
                        href: 'https://dev.iviewui.com',
                        blankTarget: true
                    },
                    {
                        title: 'Admin Plus',
                        key: '专业版',
                        href: 'https://www.iviewui.com/admin-plus',
                        blankTarget: true
                    }
                ],
                copyright: 'Copyright © 2022 View Design All Rights Reserved'
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
