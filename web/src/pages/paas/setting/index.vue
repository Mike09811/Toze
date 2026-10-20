<template>
    <div>
        <div class="i-layout-page-header">
            <PageHeader title="系统设置" content="集成授权与平台配置，后续更多设置项会放在这里。" hidden-breadcrumb />
        </div>
        <Card :bordered="false" dis-hover class="ivu-mt">
            <div slot="title">
                <Icon type="logo-github" class="ivu-mr-8" />
                GitHub 授权
            </div>
            <div slot="extra">
                <Button type="primary" icon="md-link" :disabled="!oauthConfigured" @click="handleAuth">一键授权 GitHub</Button>
                <Button icon="md-refresh" class="ivu-ml-8" @click="loadData">刷新</Button>
            </div>
            <Alert v-if="!oauthConfigured" type="warning" show-icon class="ivu-mb-16">
                <template #desc>
                    OAuth 尚未配置，暂时无法一键授权（已授权账号与公开仓库构建不受影响）。配置方法：<br />
                    1. GitHub → Settings → Developer settings → OAuth Apps → New OAuth App<br />
                    2. Homepage URL 填 <code>{{ appOrigin }}</code>，回调地址填 <code>{{ callbackUrl }}</code><br />
                    3. 把 Client ID / Client Secret 写入服务器 <code>.env</code>（GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET），重启后生效
                </template>
            </Alert>
            <Alert v-else-if="routeFromCallback" type="success" show-icon class="ivu-mb-16">
                <template #desc>授权成功，账号已加入下方列表。</template>
            </Alert>
            <Table :columns="columns" :data="accounts" :loading="loading">
                <template #default="{ row }">
                    <Tag v-if="row.is_default" color="green">默认</Tag>
                </template>
                <template #action="{ row }">
                    <div>
                        <a v-if="!row.is_default" @click="handleSetDefault(row)">设为默认</a>
                        <a v-else class="page-paas-muted" style="pointer-events: none;">使用中</a>
                        <Divider type="vertical" />
                        <a class="page-paas-danger" @click="handleDelete(row)">删除</a>
                    </div>
                </template>
            </Table>
            <div class="ivu-mt ivu-text-right page-paas-muted">
                支持授权多个账号；应用构建时可指定账号，未指定时使用默认账号。公开仓库无需授权。
            </div>
        </Card>
    </div>
</template>
<script>
    import { GetPaasGitHubStatus, GetPaasGitHubAuthURL, SetPaasGitHubDefault, DeletePaasGitHubAccount } from '@api/paas';

    export default {
        name: 'paas-setting',
        data () {
            return {
                columns: [
                    { title: 'GitHub 账号', key: 'login', minWidth: 160,
                        render: (h, { row }) => h('a', {
                            attrs: { href: `https://github.com/${row.login}`, target: '_blank', rel: 'noopener' }
                        }, row.login) },
                    { title: '默认', slot: 'default', width: 90, align: 'center' },
                    { title: '授权时间', key: 'created_at', width: 180 },
                    { title: '操作', slot: 'action', width: 160, align: 'center' }
                ],
                accounts: [],
                oauthConfigured: false,
                loading: false
            };
        },
        computed: {
            appOrigin () {
                return window.location.origin;
            },
            callbackUrl () {
                return `${window.location.origin}/api/paas/github/callback`;
            },
            routeFromCallback () {
                return this.$route.query.github === 'ok';
            }
        },
        methods: {
            loadData () {
                if (this.loading) return;
                this.loading = true;
                GetPaasGitHubStatus().then(res => {
                    this.accounts = res.accounts || [];
                    this.oauthConfigured = res.oauth_configured;
                    this.loading = false;
                }).catch(() => {
                    this.loading = false;
                });
            },
            handleAuth () {
                GetPaasGitHubAuthURL().then(res => {
                    window.location.href = res.url;
                });
            },
            handleSetDefault (row) {
                SetPaasGitHubDefault({ id: row.id }).then(() => {
                    this.$Message.success('已设为默认账号');
                    this.loadData();
                });
            },
            handleDelete (row) {
                this.$Modal.confirm({
                    title: '删除确认',
                    content: `确认删除账号 <strong>${row.login}</strong> 的授权？使用该账号构建私有仓库的应用将无法再次构建。`,
                    onOk: () => {
                        DeletePaasGitHubAccount({ id: row.id }).then(() => {
                            this.$Message.success('已删除');
                            this.loadData();
                        });
                    }
                });
            }
        },
        mounted () {
            this.loadData();
        }
    };
</script>
<style lang="less">
    .page-paas-muted {
        color: #808695;
    }
</style>
