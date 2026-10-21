<template>
    <div>
        <div class="i-layout-page-header">
            <PageHeader title="应用管理" content="把 Docker 镜像部署到纳管服务器：docker run 由平台拼接下发，修改配置后需重新部署生效。" hidden-breadcrumb />
        </div>
        <Card :bordered="false" dis-hover class="ivu-mt">
            <Button type="primary" icon="md-add" @click="handleOpenCreate">新建应用</Button>
            <Button icon="md-refresh" class="ivu-ml-8" @click="getData">刷新</Button>
            <Table
                :columns="columns"
                :data="list"
                :loading="loading"
                class="ivu-mt"
            >
                <template #ports="{ row }">{{ row.host_port }} → {{ row.container_port }}</template>
                <template #status="{ row }">
                    <Badge v-if="row.status === 1" color="green" text="运行中" />
                    <Badge v-else-if="row.status === 2" color="yellow" text="已停止" />
                    <Badge v-else-if="row.status === 3" color="blue" text="部署中" />
                    <Badge v-else color="gray" text="未部署" />
                </template>
                <template #source="{ row }">
                    <code v-if="row.source_repo">{{ row.source_repo }}#{{ row.source_branch || 'main' }}</code>
                    <code v-else>{{ row.image }}</code>
                </template>
                <template #lastError="{ row }">
                    <Tooltip v-if="row.last_error" :content="row.last_error" placement="top" max-width="400">
                        <span class="page-paas-ellipsis page-paas-error">{{ row.last_error }}</span>
                    </Tooltip>
                    <span v-else>-</span>
                </template>
                <template #action="{ row }">
                    <div>
                        <a v-if="row.status !== 1" @click="handleAction(row, 'deploy')">部署</a>
                        <a v-else @click="handleAction(row, 'restart')">重启</a>
                        <Divider type="vertical" />
                        <a v-if="row.status === 1" @click="handleAction(row, 'stop')">停止</a>
                        <a v-else-if="row.status === 2" @click="handleAction(row, 'start')">启动</a>
                        <a v-else @click="handleAction(row, 'refresh')">查状态</a>
                        <Divider type="vertical" />
                        <a @click="goDetail(row, 'runtime')">日志</a>
                        <Divider type="vertical" />
                        <a @click="goDetail(row, 'build')">构建日志</a>
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
            :title="modal.isEdit ? '编辑应用' : '新建应用'"
            width="640"
            :loading="modal.submitting"
            @on-ok="handleSubmit"
        >
            <Alert v-if="form.sourceType === 'github' && accounts.length === 0" type="warning" show-icon class="ivu-mb-16">
                <template #desc>
                    尚未授权 GitHub 账号：可直接构建公开仓库；私有仓库请先到
                    <router-link to="/paas/setting">系统设置</router-link>
                    授权。
                </template>
            </Alert>
            <Form :model="form" :rules="rules" ref="appForm" label-position="top">
                <FormItem prop="server_id" label="所属服务器">
                    <Select v-model="form.server_id" placeholder="选择纳管的服务器">
                        <Option v-for="item in servers" :key="item.id" :value="item.id">{{ item.name }}（{{ item.ip }}）</Option>
                    </Select>
                </FormItem>
                <Row :gutter="16">
                    <Col span="12">
                        <FormItem prop="name" label="应用名（即容器名）">
                            <Input v-model="form.name" placeholder="小写字母开头，如 web" :disabled="modal.isEdit" />
                        </FormItem>
                    </Col>
                    <Col span="12">
                        <FormItem label="来源">
                            <RadioGroup v-model="form.sourceType">
                                <Radio label="image">镜像</Radio>
                                <Radio label="github">GitHub 构建</Radio>
                            </RadioGroup>
                        </FormItem>
                    </Col>
                </Row>
                <Row :gutter="16" v-if="form.sourceType === 'image'">
                    <Col span="24">
                        <FormItem prop="image" label="镜像">
                            <Input v-model="form.image" placeholder="如 nginx:1.27 或 registry.com/app:v1" />
                        </FormItem>
                    </Col>
                </Row>
                <Row :gutter="16" v-if="form.sourceType === 'github'">
                    <Col span="8">
                        <FormItem label="GitHub 账号">
                            <Select v-model="form.source_account_id" placeholder="默认账号" clearable @on-change="handleAccountChange">
                                <Option v-for="item in accounts" :key="item.id" :value="item.id">
                                    {{ item.login }}{{ item.is_default ? '（默认）' : '' }}
                                </Option>
                            </Select>
                        </FormItem>
                    </Col>
                    <Col span="10">
                        <FormItem prop="source_repo" label="GitHub 仓库">
                            <Select v-model="form.source_repo" filterable placeholder="选择仓库，或直接输入 owner/repo" @on-open-change="handleRepoDropdownOpen">
                                <Option v-for="item in repos" :key="item.full_name" :value="item.full_name">
                                    {{ item.full_name }}{{ item.private ? '（私有）' : '' }}
                                </Option>
                            </Select>
                        </FormItem>
                    </Col>
                    <Col span="6">
                        <FormItem prop="source_branch" label="分支">
                            <Input v-model="form.source_branch" placeholder="默认 main" />
                        </FormItem>
                    </Col>
                </Row>
                <Row :gutter="16">
                    <Col span="8">
                        <FormItem prop="host_port" label="宿主机端口">
                            <InputNumber v-model="form.host_port" :min="1" :max="65535" style="width: 100%" />
                        </FormItem>
                    </Col>
                    <Col span="8">
                        <FormItem prop="container_port" label="容器端口">
                            <InputNumber v-model="form.container_port" :min="1" :max="65535" style="width: 100%" />
                        </FormItem>
                    </Col>
                    <Col span="8">
                        <FormItem prop="restart" label="重启策略">
                            <Select v-model="form.restart">
                                <Option value="always">always</Option>
                                <Option value="unless-stopped">unless-stopped</Option>
                                <Option value="on-failure">on-failure</Option>
                                <Option value="no">no</Option>
                            </Select>
                        </FormItem>
                    </Col>
                </Row>
                <FormItem label="环境变量（每行一个，格式 KEY=VALUE）">
                    <Input v-model="form.envText" type="textarea" :autosize="{ minRows: 3 }" placeholder="TOKEN=abc123&#10;MODE=production" />
                </FormItem>
                <FormItem label="数据卷挂载（每行一个，格式 /宿主机路径:/容器路径[:ro]）">
                    <Input v-model="form.volumeText" type="textarea" :autosize="{ minRows: 2 }" placeholder="/srv/web/data:/data&#10;/srv/web/conf:/etc/app:ro" />
                </FormItem>
            </Form>
        </Modal>

    </div>
</template>
<script>
    import { GetPaasAppList, CreatePaasApp, UpdatePaasApp, DeletePaasApp, DeployPaasApp, StartPaasApp, StopPaasApp, RestartPaasApp, RefreshPaasAppStatus, GetPaasServerList, GetPaasGitHubStatus, GetPaasGitHubRepos } from '@api/paas';

    export default {
        name: 'paas-app',
        data () {
            return {
                columns: [
                    { title: '应用', key: 'name', minWidth: 120 },
                    { title: '来源', key: 'source', slot: 'source', minWidth: 200 },
                    { title: '服务器', key: 'server_name', minWidth: 100 },
                    { title: '端口(宿主→容器)', key: 'ports', slot: 'ports', width: 130, align: 'center' },
                    { title: '状态', key: 'status', slot: 'status', width: 100, align: 'center' },
                    { title: '最近错误', key: 'last_error', slot: 'lastError', minWidth: 160 },
                    { title: '创建时间', key: 'created_at', width: 170 },
                    { title: '操作', slot: 'action', width: 280, align: 'center', fixed: 'right' }
                ],
                list: [],
                servers: [],
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
                    server_id: [{ required: true, message: '请选择服务器', trigger: 'change' }],
                    name: [
                        { required: true, message: '请输入应用名', trigger: 'blur' },
                        { pattern: /^[a-z][a-z0-9-]{0,62}$/, message: '小写字母开头，仅小写字母/数字/中划线', trigger: 'blur' }
                    ],
                    image: [{
                        validator: (rule, value, callback) => {
                            if (this.form.sourceType === 'image' && !value) callback(new Error('请输入镜像地址'));
                            else callback();
                        },
                        trigger: 'blur'
                    }],
                    source_repo: [{
                        validator: (rule, value, callback) => {
                            if (this.form.sourceType === 'github' && !value) callback(new Error('请选择或输入仓库'));
                            else callback();
                        },
                        trigger: 'change'
                    }]
                },
                accounts: [],
                repos: [],
                reposLoaded: false,
                pollTimer: null
            };
        },
        methods: {
            defaultForm () {
                return {
                    id: '',
                    server_id: '',
                    name: '',
                    sourceType: 'image',
                    image: '',
                    source_repo: '',
                    source_branch: 'main',
                    source_account_id: 0,
                    container_port: 80,
                    host_port: 8080,
                    restart: 'always',
                    envText: '',
                    volumeText: ''
                };
            },
            getData () {
                if (this.loading) return;
                this.loading = true;
                GetPaasAppList({
                    offset: (this.current - 1) * this.limit,
                    limit: this.limit
                }).then(res => {
                    this.list = res.list;
                    this.total = res.total;
                    this.loading = false;
                    this.maybePoll();
                }).catch(() => {
                    this.loading = false;
                });
            },
            // 存在「部署中」的应用时自动轮询，直到全部结束
            maybePoll () {
                const deploying = this.list.some(item => item.status === 3);
                if (deploying && !this.pollTimer) {
                    this.pollTimer = setInterval(() => this.getData(), 3000);
                } else if (!deploying && this.pollTimer) {
                    clearInterval(this.pollTimer);
                    this.pollTimer = null;
                }
            },
            handleChangePageSize (size) {
                this.current = 1;
                this.limit = size;
                this.getData();
            },
            loadServers () {
                GetPaasServerList({ offset: 0, limit: 200 }).then(res => {
                    this.servers = res.list;
                });
            },
            handleOpenCreate () {
                this.form = this.defaultForm();
                this.modal.isEdit = false;
                this.modal.show = true;
            },
            handleEdit (row) {
                this.form = {
                    id: row.id,
                    server_id: row.server_id,
                    name: row.name,
                    sourceType: row.source_repo ? 'github' : 'image',
                    image: row.image,
                    source_repo: row.source_repo || '',
                    source_branch: row.source_branch || 'main',
                    source_account_id: row.source_account_id || 0,
                    container_port: row.container_port,
                    host_port: row.host_port,
                    restart: row.restart || 'always',
                    envText: (row.env || []).join('\n'),
                    volumeText: (row.volumes || []).join('\n')
                };
                this.modal.isEdit = true;
                this.modal.show = true;
            },
            handleSubmit () {
                this.$refs.appForm.validate(valid => {
                    if (!valid) {
                        this.modal.submitting = false;
                        return;
                    }
                    const payload = {
                        id: this.form.id,
                        server_id: this.form.server_id,
                        name: this.form.name,
                        image: this.form.sourceType === 'image' ? this.form.image : '',
                        source_repo: this.form.sourceType === 'github' ? this.form.source_repo : '',
                        source_branch: this.form.sourceType === 'github' ? this.form.source_branch : '',
                        source_account_id: this.form.sourceType === 'github' ? (this.form.source_account_id || '') : '',
                        container_port: this.form.container_port,
                        host_port: this.form.host_port,
                        restart: this.form.restart,
                        env: this.splitLines(this.form.envText),
                        volumes: this.splitLines(this.form.volumeText)
                    };
                    const action = this.modal.isEdit ? UpdatePaasApp : CreatePaasApp;
                    action(payload).then(() => {
                        this.modal.show = false;
                        this.$Message.success(this.modal.isEdit ? '保存成功，重新部署后生效' : '创建成功');
                        this.getData();
                    }).catch(() => {
                        this.modal.submitting = false;
                    });
                });
            },
            splitLines (text) {
                return (text || '').split('\n').map(item => item.trim()).filter(item => item !== '');
            },
            handleAction (row, action) {
                const api = { deploy: DeployPaasApp, start: StartPaasApp, stop: StopPaasApp, restart: RestartPaasApp, refresh: RefreshPaasAppStatus }[action];
                if (action === 'deploy') {
                    this.$Modal.confirm({
                        title: '部署确认',
                        content: `将在 <strong>${row.server_name}</strong> 上（重新）部署 <strong>${row.name}</strong>，旧容器会被移除。镜像拉取可能需要几分钟。`,
                        onOk: () => {
                            api({ id: row.id }).then(() => {
                                this.$Message.success('部署已开始');
                                this.getData();
                                this.goDetail(row, 'build');
                            });
                        }
                    });
                    return;
                }
                api({ id: row.id }).then(() => {
                    this.$Message.success('操作成功');
                    this.getData();
                });
            },
            goDetail (row, tab) {
                this.$router.push({ name: 'paas-app-detail', params: { id: row.id }, query: { tab } });
            },
            loadGitHubStatus () {
                GetPaasGitHubStatus().then(res => {
                    this.accounts = res.accounts || [];
                });
            },
            handleAccountChange () {
                this.repos = [];
                this.reposLoaded = false;
            },
            handleRepoDropdownOpen (open) {
                if (open && !this.reposLoaded && this.accounts.length > 0) {
                    GetPaasGitHubRepos({ account_id: this.form.source_account_id || '' }).then(res => {
                        this.repos = res || [];
                        this.reposLoaded = true;
                    });
                }
            },
            handleDelete (row) {
                this.$Modal.confirm({
                    title: '删除确认',
                    content: `确认删除应用 <strong>${row.name}</strong>？远程容器、域名绑定与 Caddy 配置会一并清理。`,
                    onOk: () => {
                        DeletePaasApp({ id: row.id }).then(() => {
                            this.$Message.success('删除成功');
                            this.getData();
                        });
                    }
                });
            }
        },
        mounted () {
            this.getData();
            this.loadServers();
            this.loadGitHubStatus();

        },
        beforeUnmount () {
            if (this.pollTimer) clearInterval(this.pollTimer);
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
