<template>
    <div class="page-account">
        <div v-if="showI18n" class="page-account-header">
            <i-header-i18n outside />
        </div>
        <div class="page-account-container">
            <div class="page-account-top">
                <div class="page-account-top-logo">
                    <img src="@/assets/images/logo.png" alt="logo">
                </div>
                <div class="page-account-top-desc">自带服务器的极简 PaaS</div>
            </div>
            <Login @on-submit="handleSubmit">
                <UserName name="username" />
                <Password name="password" enter-to-submit />
                <div class="page-account-auto-login">
                    <Checkbox v-model="autoLogin" size="large">{{ $t('page.login.remember') }}</Checkbox>
                </div>
                <Submit>{{ $t('page.login.submit') }}</Submit>
            </Login>
            <div class="page-account-other">
                <router-link class="page-account-register" :to="{ name: 'register' }">{{ $t('page.login.signup') }}</router-link>
            </div>
        </div>
        <i-copyright />
    </div>
</template>
<script>
    import iCopyright from '@/components/copyright';
    import { mapActions } from 'vuex';
    import mixins from '../mixins';

    export default {
        name: 'page-account-login',
        mixins: [ mixins ],
        components: { iCopyright },
        data () {
            return {
                autoLogin: true
            }
        },
        methods: {
            ...mapActions('admin/account', [
                'login'
            ]),
            /**
             * @description 登录
             * 表单校验已有 View UI Plus 自动完成，如有需要修改，请阅读 View UI Plus 文档
             */
            handleSubmit (valid, values) {
                if (valid) {
                    const { username, password } = values;
                    this.login({
                        username,
                        password
                    })
                        .then(() => {
                            // 重定向对象不存在则返回顶层路径
                            this.$router.replace(this.$route.query.redirect || '/', () => {});
                    });
                }
            }
        }
    };
</script>
<style lang="less">
    @import "../account";
</style>
