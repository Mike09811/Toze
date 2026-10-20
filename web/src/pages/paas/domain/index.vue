<template>
    <div>
        <div class="i-layout-page-header">
            <PageHeader title="域名管理" content="把域名绑定到应用：平台自动生成 Caddy 反向代理配置，HTTPS 证书由 Caddy 自动申请与续期。" hidden-breadcrumb />
        </div>
        <Card :bordered="false" dis-hover class="ivu-mt">
            <Button type="primary" icon="md-add" @click="handleOpenCreate">绑定域名</Button>
            <Button icon="md-refresh" class="ivu-ml-8" @click="getData">刷新</Button>
            <Table
                :columns="columns"
                :data="list"
                :loading="loading"
                class="ivu-mt"
            >
                <template #domain="{ row }">
                    <a :href="'https://' + row.domain" target="_blank" rel="noopener">{{ row.domain }}</a>
                </template>
                <template #action="{ row }">
                    <div>
                        <a class="page-paas-danger" @click="handleDelete(row)">解绑</a>
                    </div>
                </template>
            </Table>
            <div class="ivu-mt ivu-text-center">
                <Page
                    :total="total"
                    v-model="current"
                    show-total
                    show-sizer
                    :page-size="limit"
                    @on-change="getData"
                    @on-page-size-change="handleChangePageSize"
                />
            </div>
        </Card>

        <Modal
            v-model="modal.show"
            title="绑定域名"
            :loading="modal.submitting"
            @on-ok="handleSubmit"
        >
            <Form :model="form" :rules="rules" ref="domainForm" label-position="top">
                <FormItem prop="app_id" label="目标应用">
                    <Select v-model="form.app_id" placeholder="选择要绑定的应用">
                        <Option v-for="item in apps" :key="item.id" :value="item.id">
                            {{ item.name }}（{{ item.server_name }} → 端口 {{ item.host_port }}）
                        </Option>
                    </Select>
                </FormItem>
                <FormItem prop="domain" label="域名">
                    <Input v-model="form.domain" placeholder="如 app.example.com" />
                </FormItem>
                <Alert type="info" show-icon>
                    绑定前请先把域名 A 记录解析到应用所在服务器 IP，并确保服务器 80 / 443 端口开放。
                </Alert>
            </Form>
        </Modal>
    </div>
</template>
<script>
    import { GetPaasDomainList, CreatePaasDomain, DeletePaasDomain, GetPaasAppList } from '@api/paas';

    export default {
        name: 'paas-domain',
        data () {
            return {
                columns: [
                    { title: '域名', key: 'domain', slot: 'domain', minWidth: 220 },
                    { title: '应用', key: 'app_name', minWidth: 120 },
                    { title: '服务器', key: 'server_name', minWidth: 120 },
                    { title: '反代端口', key: 'host_port', width: 100, align: 'center' },
                    { title: '绑定时间', key: 'created_at', width: 170 },
                    { title: '操作', slot: 'action', width: 100, align: 'center' }
                ],
                list: [],
                apps: [],
                loading: false,
                current: 1,
                limit: 10,
                total: 0,
                modal: {
                    show: false,
                    submitting: false
                },
                form: {
                    app_id: '',
                    domain: ''
                },
                rules: {
                    app_id: [{ required: true, message: '请选择应用', trigger: 'change' }],
                    domain: [
                        { required: true, message: '请输入域名', trigger: 'blur' },
                        { pattern: /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$/, message: '域名格式不正确', trigger: 'blur' }
                    ]
                }
            };
        },
        methods: {
            getData () {
                if (this.loading) return;
                this.loading = true;
                GetPaasDomainList({
                    offset: (this.current - 1) * this.limit,
                    limit: this.limit
                }).then(res => {
                    this.list = res.list;
                    this.total = res.total;
                    this.loading = false;
                }).catch(() => {
                    this.loading = false;
                });
            },
            handleChangePageSize (size) {
                this.current = 1;
                this.limit = size;
                this.getData();
            },
            loadApps () {
                GetPaasAppList({ offset: 0, limit: 200 }).then(res => {
                    this.apps = res.list;
                });
            },
            handleOpenCreate () {
                this.form = { app_id: '', domain: '' };
                this.loadApps();
                this.modal.show = true;
            },
            handleSubmit () {
                this.$refs.domainForm.validate(valid => {
                    if (!valid) {
                        this.modal.submitting = false;
                        return;
                    }
                    CreatePaasDomain(this.form).then(() => {
                        this.modal.show = false;
                        this.$Message.success('绑定成功，证书稍后由 Caddy 自动签发');
                        this.getData();
                    }).catch(() => {
                        this.modal.submitting = false;
                    });
                });
            },
            handleDelete (row) {
                this.$Modal.confirm({
                    title: '解绑确认',
                    content: `确认解绑 <strong>${row.domain}</strong>？Caddy 反代配置会同步移除，DNS 解析需自行处理。`,
                    onOk: () => {
                        DeletePaasDomain({ id: row.id }).then(() => {
                            this.$Message.success('解绑成功');
                            this.getData();
                        });
                    }
                });
            }
        },
        mounted () {
            this.getData();
        }
    };
</script>
<style lang="less">
    .page-paas-danger {
        color: #ed4014;
    }
</style>
