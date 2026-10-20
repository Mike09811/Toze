<template>
    <div v-resize="handleResize">
        <base-info />
        <Row :gutter="24">
            <Col :xl="18" :lg="24" :md="24" :sm="24" :xs="24" class="ivu-mb">
                <Card :bordered="false" dis-hover v-height="480">
                    <template #title>
                        <Avatar icon="ios-pulse" size="small" v-color="'#1890ff'" v-bg-color="'#e6f7ff'" />
                        <span class="ivu-pl-8">最近1小时访问情况</span>
                    </template>
                    <visit-chart ref="visitChart" />
                </Card>
            </Col>
            <Col :xl="6" :lg="24" :md="24" :sm="24" :xs="24" class="ivu-mb">
                <Row :gutter="24">
                    <Col :xl="24" :lg="12" :md="12" :sm="12" :xs="24" class="ivu-mb">
                        <Card :bordered="false" dis-hover v-height="232">
                            <template #title>
                                <Avatar icon="md-bulb" size="small" v-color="'#fa8c16'" v-bg-color="'#fff7e6'" />
                                <span class="ivu-pl-8">当前在线</span>
                            </template>
                            <visit-user />
                        </Card>
                    </Col>
                    <Col :xl="24" :lg="12" :md="12" :sm="12" :xs="24" class="ivu-mb">
                        <Card :bordered="false" dis-hover v-height="232">
                            <template #title>
                                <Avatar icon="ios-cafe" size="small" v-color="'#eb2f96'" v-bg-color="'#fff0f6'" />
                                <span class="ivu-pl-8">当前活跃度</span>
                            </template>
                            <visit-hot ref="visitHot" />
                        </Card>
                    </Col>
                </Row>
            </Col>
        </Row>
        <Row :gutter="24">
            <Col :xl="12" :lg="24" :md="24" :sm="24" :xs="24" class="ivu-mb">
                <Card :bordered="false" dis-hover v-height="300">
                    <template #title>
                        <Avatar icon="md-thumbs-up" size="small" v-color="'#ffc53d'" v-bg-color="'#fffbe6'" />
                        <span class="ivu-pl-8">平均评价</span>
                    </template>
                    <avg-rate />
                </Card>
            </Col>
            <Col :xl="6" :lg="12" :md="12" :sm="12" :xs="24" class="ivu-mb">
                <Card :bordered="false" dis-hover v-height="300">
                    <template #title>
                        <Avatar icon="md-happy" size="small" v-color="'#597ef7'" v-bg-color="'#f0f5ff'" />
                        <span class="ivu-pl-8">客户满意度</span>
                    </template>
                    <sentiment />
                </Card>
            </Col>
            <Col :xl="6" :lg="12" :md="12" :sm="12" :xs="24" class="ivu-mb">
                <Card :bordered="false" dis-hover v-height="300">
                    <template #title>
                        <Avatar icon="md-done-all" size="small" v-color="'#ff4d4f'" v-bg-color="'#fff1f0'" />
                        <span class="ivu-pl-8">本月目标</span>
                    </template>
                    <goal />
                </Card>
            </Col>
        </Row>
    </div>
</template>
<script>
    import baseInfo from './base-info';
    import visitChart from './visit-chart';
    import visitUser from './visit-user';
    import visitHot from './visit-hot';
    import avgRate from './avg-rate';
    import sentiment from './sentiment';
    import goal from './goal';

    export default {
        name: 'dashboard-monitor',
        components: { baseInfo, visitChart, visitUser, visitHot, avgRate, sentiment, goal},
        data () {
            return {

            }
        },
        methods: {
            handleResize () {
                this.$refs.visitChart.handleResize();
            }
        }
    }
</script>
