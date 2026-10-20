<template>
    <div class="page-account">
        <div class="page-account-container page-account-container-result">
            <div class="page-account-top">
                <div class="page-account-top-logo">
                    <img src="@/assets/images/logo.png" alt="logo">
                </div>
                <div class="page-account-top-desc">Admin Plus 企业级中后台前端解决方案</div>
            </div>
            <Result type="success" :title="title" :desc="desc">
                <template #actions>
                    <div>
                        <Button size="large" type="primary" to="https://mail.qq.com" target="_blank">查看邮箱</Button>
                        <Button size="large" to="/">返回首页</Button>
                    </div>
                </template>
            </Result>
        </div>
        <i-copyright />
    </div>
</template>
<script>
    import iCopyright from '@/components/copyright';

    export default {
        name: 'page-account-result',
        components: { iCopyright },
        data () {
            return {
                title: '你的账户：admin 注册成功',
                desc: '激活邮件已发送到你的邮箱中，邮件有效期为24小时。请及时登录邮箱，点击邮件中的链接激活帐户。'
            }
        }
    };
</script>
<style lang="less">
@import "../../account";
</style>
