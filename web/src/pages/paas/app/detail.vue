<template>
    <div class="page-paas-detail">
        <div class="i-layout-page-header">
            <PageHeader :title="`应用详情 - ${app.name || '...'}`" hidden-breadcrumb>
                <div slot="content" style="display: flex; align-items: center; gap: 12px;">
                    <Badge v-if="app.status === 1" color="green" text="运行中" />
                    <Badge v-else-if="app.status === 2" color="yellow" text="已停止" />
                    <Badge v-else-if="app.status === 3" color="blue" text="部署中" />
                    <Badge v-else color="gray" text="未部署" />
                    <span>修改配置请返回列表页编辑</span>
                </div>
            </PageHeader>
        </div>
        <Card :bordered="false" dis-hover class="ivu-mt">
            <div slot="extra">
                <Button type="primary" icon="md-rocket" :disabled="app.status === 3" @click="handleDeploy">部署</Button>
                <Button icon="md-refresh" class="ivu-ml-8" @click="loadApp">刷新</Button>
                <Button icon="md-arrow-back" class="ivu-ml-8" @click="$router.push({ name: 'paas-app' })">返回列表</Button>
            </div>
            <Tabs :value="activeTab" @on-click="handleTabChange">
                <TabPane label="基本信息" name="info">
                    <Row :gutter="16" class="ivu-mt-8">
                        <Col span="12" v-for="item in infoItems" :key="item.label">
                            <div class="page-paas-detail-item">
                                <div class="page-paas-detail-label">{{ item.label }}</div>
                                <div class="page-paas-detail-value">
                                    <code v-if="item.code">{{ item.value }}</code>
                                    <template v-else>{{ item.value }}</template>
                                </div>
                            </div>
                        </Col>
                        <Col span="12">
                            <div class="page-paas-detail-item">
                                <div class="page-paas-detail-label">环境变量</div>
                                <div class="page-paas-detail-value">
                                    <div v-if="app.env && app.env.length"><code v-for="e in app.env" :key="e" class="page-paas-detail-kv">{{ e }}</code></div>
                                    <span v-else>-</span>
                                </div>
                            </div>
                        </Col>
                        <Col span="12">
                            <div class="page-paas-detail-item">
                                <div class="page-paas-detail-label">数据卷</div>
                                <div class="page-paas-detail-value">
                                    <div v-if="app.volumes && app.volumes.length"><code v-for="v in app.volumes" :key="v" class="page-paas-detail-kv">{{ v }}</code></div>
                                    <span v-else>-</span>
                                </div>
                            </div>
                        </Col>
                    </Row>
                    <Alert type="error" class="ivu-mt-16" v-if="app.last_error">
                        <template #desc><b>最近错误：</b>{{ app.last_error }}</template>
                    </Alert>
                </TabPane>
                <TabPane label="构建日志" name="build">
                    <div class="page-paas-detail-toolbar">
                        <Tag v-if="buildStatus === 3" color="blue">构建中…（每 2 秒自动刷新）</Tag>
                        <Tag v-else-if="buildStatus === 1" color="green">部署成功，运行中</Tag>
                        <Tag v-else-if="buildStatus === 0" color="red">未部署 / 上次构建失败</Tag>
                        <Tag v-else color="gray">已停止</Tag>
                        <span class="page-paas-detail-meta">全量输出 {{ buildLogSize }} 字符</span>
                        <Checkbox v-model="autoScroll" class="ivu-ml-16">自动滚动</Checkbox>
                        <Button size="small" icon="md-refresh" style="margin-left: auto;" @click="loadBuildLog">刷新</Button>
                    </div>
                    <div class="page-paas-logs page-paas-detail-logs"><pre ref="buildPre">{{ buildLog || '暂无构建日志' }}</pre></div>
                </TabPane>
                <TabPane label="运行日志" name="runtime">
                    <div class="page-paas-detail-toolbar">
                        <Select v-model="runtimeTail" style="width: 140px;" @on-change="loadRuntimeLog">
                            <Option :value="200">最近 200 行</Option>
                            <Option :value="1000">最近 1000 行</Option>
                            <Option :value="5000">最近 5000 行</Option>
                        </Select>
                        <Checkbox v-model="autoScrollRuntime" class="ivu-ml-16">自动滚动</Checkbox>
                        <Button size="small" icon="md-refresh" style="margin-left: auto;" :loading="runtimeLoading" @click="loadRuntimeLog">刷新</Button>
                    </div>
                    <div class="page-paas-logs page-paas-detail-logs"><pre ref="runtimePre">{{ runtimeLog || '暂无运行日志（应用需处于部署状态）' }}</pre></div>
                </TabPane>
            </Tabs>
        </Card>
    </div>
</template>
<script>
    import { GetPaasAppList, DeployPaasApp, GetPaasAppDeployLog, GetPaasAppLogs } from '@api/paas';

    export default {
        name: 'paas-app-detail',
        data () {
            return {
                appId: this.$route.params.id,
                activeTab: this.$route.query.tab === 'runtime' ? 'runtime' : (this.$route.query.tab === 'build' ? 'build' : 'info'),
                app: {},
                buildLog: '',
                buildStatus: 0,
                buildTimer: null,
                runtimeLog: '',
                runtimeTail: 1000,
                runtimeLoading: false,
                autoScroll: true,
                autoScrollRuntime: true
            };
        },
        computed: {
            buildLogSize () {
                return (this.buildLog || '').length.toLocaleString();
            },
            infoItems () {
                const a = this.app || {};
                const source = a.source_repo ? `${a.source_repo}#${a.source_branch || 'main'}` : (a.image || '-');
                return [
                    { label: '应用名', value: a.name || '-' },
                    { label: '服务器', value: a.server_name ? `${a.server_name}（ID ${a.server_id}）` : '-' },
                    { label: '来源', value: source, code: true },
                    { label: '端口映射', value: a.host_port ? `${a.host_port} → ${a.container_port}` : '-' },
                    { label: '重启策略', value: a.restart || '-' },
                    { label: '容器 ID', value: a.container_id || '-' }
                ];
            }
        },
        methods: {
            loadApp () {
                GetPaasAppList({ offset: 0, limit: 200 }).then(res => {
                    const found = (res.list || []).find(item => item.id === this.appId);
                    if (!found) {
                        this.$Message.error('应用不存在');
                        this.$router.push({ name: 'paas-app' });
                        return;
                    }
                    this.app = found;
                    this.buildStatus = found.status;
                });
            },
            handleTabChange (tab) {
                if (tab === 'build') this.startBuildPoll();
                else this.stopBuildPoll();
                if (tab === 'runtime' && !this.runtimeLog) this.loadRuntimeLog();
            },
            loadBuildLog () {
                GetPaasAppDeployLog({ id: this.appId }).then(res => {
                    this.buildStatus = res.status;
                    this.buildLog = res.log || '';
                    this.scrollBottom('buildPre');
                    if (res.status !== 3) this.stopBuildPoll();
                });
            },
            startBuildPoll () {
                this.stopBuildPoll();
                this.loadBuildLog();
                this.buildTimer = setInterval(() => {
                    if (this.buildStatus === 3) this.loadBuildLog();
                    else this.stopBuildPoll();
                }, 2000);
            },
            stopBuildPoll () {
                if (this.buildTimer) {
                    clearInterval(this.buildTimer);
                    this.buildTimer = null;
                }
            },
            loadRuntimeLog () {
                this.runtimeLoading = true;
                GetPaasAppLogs({ id: this.appId, tail: this.runtimeTail }).then(res => {
                    this.runtimeLog = res.logs || '';
                    this.runtimeLoading = false;
                    this.scrollBottom('runtimePre');
                }).catch(() => {
                    this.runtimeLoading = false;
                });
            },
            scrollBottom (ref) {
                if (!(ref === 'buildPre' ? this.autoScroll : this.autoScrollRuntime)) return;
                this.$nextTick(() => {
                    const pre = this.$refs[ref];
                    if (pre && pre.parentElement) pre.parentElement.scrollTop = pre.parentElement.scrollHeight;
                });
            },
            handleDeploy () {
                this.$Modal.confirm({
                    title: '部署确认',
                    content: `将在 <strong>${this.app.server_name}</strong> 上（重新）部署 <strong>${this.app.name}</strong>，旧容器会被移除。`,
                    onOk: () => {
                        DeployPaasApp({ id: this.appId }).then(() => {
                            this.$Message.success('部署已开始');
                            this.buildStatus = 3;
                            this.activeTab = 'build';
                            this.startBuildPoll();
                            this.loadApp();
                        });
                    }
                });
            }
        },
        mounted () {
            this.loadApp();
            if (this.activeTab === 'build') this.startBuildPoll();
            if (this.activeTab === 'runtime') this.loadRuntimeLog();
        },
        beforeUnmount () {
            this.stopBuildPoll();
        }
    };
</script>
<style lang="less">
    .page-paas-detail {
        &-logs {
            height: calc(100vh - 320px);
            min-height: 320px;
        }
        &-toolbar {
            display: flex;
            align-items: center;
            margin-bottom: 8px;
        }
        &-meta {
            color: #808695;
            font-size: 12px;
            margin-left: 12px;
        }
        &-item {
            padding: 10px 12px;
            border: 1px solid #e8eaec;
            border-radius: 4px;
            margin-bottom: 16px;
        }
        &-label {
            color: #808695;
            font-size: 12px;
            margin-bottom: 4px;
        }
        &-value {
            word-break: break-all;
        }
        &-kv {
            display: inline-block;
            margin: 0 8px 4px 0;
            padding: 1px 6px;
            background: #f5f7f9;
            border-radius: 3px;
        }
    }
</style>
