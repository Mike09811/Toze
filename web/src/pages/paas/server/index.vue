<template>
    <div>
        <div class="i-layout-page-header">
            <PageHeader title="服务器管理" content="纳管自有 VPS：SSH 直连下发指令，服务器侧只需 Docker + Caddy，不安装常驻 Agent。" hidden-breadcrumb />
        </div>
        <Card :bordered="false" dis-hover class="ivu-mt">
            <Button type="primary" icon="md-add" @click="handleOpenCreate">添加服务器</Button>
            <Button icon="md-refresh" class="ivu-ml-8" @click="getData">刷新</Button>
            <Table
                :columns="columns"
                :data="list"
                :loading="loading"
                class="ivu-mt"
            >
                <template #authType="{ row }">
                    <Tag v-if="row.auth_type === 1" color="blue">密码</Tag>
                    <Tag v-else color="cyan">私钥</Tag>
                </template>
                <template #status="{ row }">
                    <Badge v-if="row.status === 1" color="green" text="在线" />
                    <Badge v-else-if="row.status === 2" color="blue" text="初始化中" />
                    <Badge v-else color="red" text="离线" />
                </template>
                <template #ssh="{ row }">{{ row.ssh_user }}@{{ row.ip }}:{{ row.ssh_port }}</template>
                <template #lastError="{ row }">
                    <Tooltip v-if="row.last_error" :content="row.last_error" placement="top" max-width="400">
                        <span class="page-paas-ellipsis page-paas-error">{{ row.last_error }}</span>
                    </Tooltip>
                    <span v-else>-</span>
                </template>
                <template #action="{ row }">
                    <div>
                        <a @click="handleTest(row)">测试</a>
                        <Divider type="vertical" />
                        <a @click="handleInit(row)">初始化</a>
                        <Divider type="vertical" />
                        <a @click="handleEdit(row)">编辑</a>
                        <Divider type="vertical" />
                        <a class="page-paas-danger" @click="handleDelete(row)">删除</a>
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
            :title="modal.isEdit ? '编辑服务器' : '添加服务器'"
            :loading="modal.submitting"
            @on-ok="handleSubmit"
        >
            <Form :model="form" :rules="rules" ref="serverForm" label-position="top">
                <FormItem prop="name" label="名称">
                    <Input v-model="form.name" placeholder="服务器备注名称，如 hk-vps-1" />
                </FormItem>
                <FormItem prop="ip" label="公网 IP">
                    <Input v-model="form.ip" placeholder="服务器公网 IP" />
                </FormItem>
                <Row :gutter="16">
                    <Col span="12">
                        <FormItem prop="ssh_port" label="SSH 端口">
                            <InputNumber v-model="form.ssh_port" :min="1" :max="65535" style="width: 100%" />
                        </FormItem>
                    </Col>
                    <Col span="12">
                        <FormItem prop="ssh_user" label="SSH 用户">
                            <Input v-model="form.ssh_user" placeholder="默认 root" />
                        </FormItem>
                    </Col>
                </Row>
                <FormItem prop="auth_type" label="认证方式">
                    <RadioGroup v-model="form.auth_type">
                        <Radio :label="1">密码</Radio>
                        <Radio :label="2">私钥</Radio>
                    </RadioGroup>
                </FormItem>
                <FormItem prop="auth_content" :label="modal.isEdit ? '凭证（留空保持不变）' : 'SSH 密码 / 私钥内容'">
                    <Input
                        v-if="form.auth_type === 1"
                        v-model="form.auth_content"
                        type="password"
                        password
                        placeholder="SSH 登录密码"
                    />
                    <Input
                        v-else
                        v-model="form.auth_content"
                        type="textarea"
                        :autosize="{ minRows: 4 }"
                        placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
                    />
                </FormItem>
                <Alert type="info" show-icon>凭证使用 AES-GCM 加密存储，保存后任何接口都不会回传明文。首次使用请确认服务端已配置 PAAS_AES_KEY。</Alert>
            </Form>
        </Modal>
    </div>
</template>
<script>
    import { GetPaasServerList, CreatePaasServer, UpdatePaasServer, DeletePaasServer, TestPaasServer, InitPaasServer } from '@api/paas';

    export default {
        name: 'paas-server',
        data () {
            return {
                columns: [
                    { title: '名称', key: 'name', minWidth: 120 },
                    { title: 'SSH', key: 'ssh', slot: 'ssh', minWidth: 200 },
                    { title: '认证', key: 'auth_type', slot: 'authType', width: 90, align: 'center' },
                    { title: '状态', key: 'status', slot: 'status', width: 110, align: 'center' },
                    { title: 'Docker', key: 'docker_ver', width: 110 },
                    { title: '应用数', key: 'app_count', width: 80, align: 'center' },
                    { title: '最近错误', key: 'last_error', slot: 'lastError', minWidth: 160 },
                    { title: '添加时间', key: 'created_at', width: 170 },
                    { title: '操作', slot: 'action', width: 220, align: 'center', fixed: 'right' }
                ],
                list: [],
                loading: false,
                current: 1,
                limit: 10,
                total: 0,
                modal: {
                    show: false,
                    isEdit: false,
                    submitting: false
                },
                form: this.defaultForm(),
                rules: {
                    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
                    ip: [{ required: true, message: '请输入公网 IP', trigger: 'blur' }],
                    auth_content: [{
                        validator: (rule, value, callback) => {
                            if (!this.modal.isEdit && !value) callback(new Error('请输入凭证'));
                            else callback();
                        },
                        trigger: 'blur'
                    }]
                }
            };
        },
        methods: {
            defaultForm () {
                return {
                    id: '',
                    name: '',
                    ip: '',
                    ssh_port: 22,
                    ssh_user: 'root',
                    auth_type: 1,
                    auth_content: ''
                };
            },
            getData () {
                if (this.loading) return;
                this.loading = true;
                GetPaasServerList({
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
            handleOpenCreate () {
                this.form = this.defaultForm();
                this.modal.isEdit = false;
                this.modal.show = true;
            },
            handleEdit (row) {
                this.form = {
                    id: row.id,
                    name: row.name,
                    ip: row.ip,
                    ssh_port: row.ssh_port,
                    ssh_user: row.ssh_user,
                    auth_type: row.auth_type,
                    auth_content: ''
                };
                this.modal.isEdit = true;
                this.modal.show = true;
            },
            handleSubmit () {
                this.$refs.serverForm.validate(valid => {
                    if (!valid) {
                        this.modal.submitting = false;
                        return;
                    }
                    const action = this.modal.isEdit ? UpdatePaasServer : CreatePaasServer;
                    action(this.form).then(() => {
                        this.modal.show = false;
                        this.$Message.success(this.modal.isEdit ? '保存成功' : '添加成功');
                        this.getData();
                    }).catch(() => {
                        this.modal.submitting = false;
                    });
                });
            },
            handleTest (row) {
                TestPaasServer({ id: row.id }).then(res => {
                    this.$Message.success(`连接成功，Docker ${res.docker_ver || '未安装'}`);
                    this.getData();
                });
            },
            handleInit (row) {
                this.$Modal.confirm({
                    title: '初始化确认',
                    content: `将在 <strong>${row.name}</strong> 上安装 Docker 与 Caddy 并设置开机自启（Debian/Ubuntu），可能需要几分钟，期间请勿重复操作。继续？`,
                    onOk: () => {
                        InitPaasServer({ id: row.id }).then(() => {
                            this.$Message.success('初始化已开始，稍后点击刷新查看结果');
                            this.getData();
                        });
                    }
                });
            },
            handleDelete (row) {
                this.$Modal.confirm({
                    title: '删除确认',
                    content: `确认删除服务器 <strong>${row.name}（${row.ip}）</strong>？服务器下存在应用时将无法删除。`,
                    onOk: () => {
                        DeletePaasServer({ id: row.id }).then(() => {
                            this.$Message.success('删除成功');
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
    .page-paas {
        &-ellipsis {
            display: inline-block;
            max-width: 100%;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            vertical-align: bottom;
        }
        &-error {
            color: #ed4014;
        }
        &-danger {
            color: #ed4014;
        }
        &-logs {
            max-height: 420px;
            overflow: auto;
            padding: 8px 12px;
            background: #17233d;
            color: #c5c8ce;
            border-radius: 4px;
            pre {
                margin: 0;
                white-space: pre-wrap;
                word-break: break-all;
                font-family: Menlo, Consolas, monospace;
                font-size: 12px;
                line-height: 1.6;
            }
        }
    }
</style>
