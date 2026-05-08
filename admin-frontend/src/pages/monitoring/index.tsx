import React, { useEffect, useState } from 'react';
import { PageContainer, ProTable, ProColumns } from '@ant-design/pro-components';
import {
  Button,
  Card,
  Row,
  Col,
  Statistic,
  Tag,
  Space,
  Tabs,
  Alert,
  Table,
  Badge,
} from 'antd';
import {
  ReloadOutlined,
  DashboardOutlined,
  AlertOutlined,
  BugOutlined,
  TeamOutlined,
  CloudOutlined,
} from '@ant-design/icons';
import type { ConcurrencyStats, RealtimeTraffic, ErrorLog, AlertRule } from '@/services/api';
import { getConcurrencyStats, getRealtimeTraffic, getErrorLogs, getAlertRules } from '@/services/api';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import dayjs from 'dayjs';

const { TabPane } = Tabs;

const MonitoringPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [concurrencyStats, setConcurrencyStats] = useState<ConcurrencyStats | null>(null);
  const [trafficData, setTrafficData] = useState<RealtimeTraffic[]>([]);
  const [errorLogs, setErrorLogs] = useState<ErrorLog[]>([]);
  const [alertRules, setAlertRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchAllData();
    const interval = setInterval(fetchRealtimeTraffic, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchAllData = async () => {
    setLoading(true);
    await Promise.all([
      fetchConcurrencyStats(),
      fetchRealtimeTraffic(),
      fetchErrorLogs(),
      fetchAlertRules(),
    ]);
    setLoading(false);
  };

  const fetchConcurrencyStats = async () => {
    try {
      const response = await getConcurrencyStats();
      if (response.success) {
        setConcurrencyStats(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch concurrency stats:', error);
    }
  };

  const fetchRealtimeTraffic = async () => {
    try {
      const response = await getRealtimeTraffic({ duration: 60 });
      if (response.success) {
        setTrafficData(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch realtime traffic:', error);
    }
  };

  const fetchErrorLogs = async () => {
    try {
      const response = await getErrorLogs({ current: 1, pageSize: 100 });
      if (response.success) {
        setErrorLogs(response.data.list);
      }
    } catch (error) {
      console.error('Failed to fetch error logs:', error);
    }
  };

  const fetchAlertRules = async () => {
    try {
      const response = await getAlertRules({ current: 1, pageSize: 100 });
      if (response.success) {
        setAlertRules(response.data.list);
      }
    } catch (error) {
      console.error('Failed to fetch alert rules:', error);
    }
  };

  // QPS 实时监控图表
  const qpsOption: EChartsOption = {
    title: { text: 'QPS 实时监控', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: trafficData.map(item => dayjs(item.timestamp).format('HH:mm:ss')),
    },
    yAxis: { type: 'value', name: 'QPS' },
    series: [{
      name: 'QPS',
      type: 'line',
      smooth: true,
      areaStyle: { opacity: 0.3 },
      data: trafficData.map(item => item.qps),
      itemStyle: { color: '#1890ff' },
    }],
  };

  // 延迟趋势图表
  const latencyOption: EChartsOption = {
    title: { text: '延迟趋势', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: trafficData.map(item => dayjs(item.timestamp).format('HH:mm:ss')),
    },
    yAxis: { type: 'value', name: 'ms' },
    series: [{
      name: '延迟',
      type: 'line',
      smooth: true,
      data: trafficData.map(item => item.latency),
      itemStyle: { color: '#52c41a' },
    }],
  };

  // 错误率趋势图表
  const errorRateOption: EChartsOption = {
    title: { text: '错误率趋势', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: trafficData.map(item => dayjs(item.timestamp).format('HH:mm:ss')),
    },
    yAxis: { type: 'value', name: '%', max: 100 },
    series: [{
      name: '错误率',
      type: 'line',
      data: trafficData.map(item => item.errorRate),
      itemStyle: { color: '#ff4d4f' },
      markLine: {
        data: [{ yAxis: 5, name: '告警阈值' }],
        lineStyle: { color: '#ff4d4f', type: 'dashed' },
      },
    }],
  };

  // 并发分布图表
  const concurrencyByProviderOption: EChartsOption = {
    title: { text: '按供应商分布', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'item' },
    series: [{
      name: '并发数',
      type: 'pie',
      radius: ['40%', '70%'],
      data: concurrencyStats?.byProvider.map(item => ({ name: item.provider, value: item.count })) || [],
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
    }],
  };

  const errorLogColumns: ProColumns<ErrorLog>[] = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 170,
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 100,
      render: (level: string) => {
        const levelMap: Record<string, { color: string; text: string }> = {
          error: { color: 'error', text: '错误' },
          warning: { color: 'warning', text: '警告' },
          critical: { color: 'red', text: '严重' },
        };
        const item = levelMap[level] || { color: 'default', text: level };
        return <Badge status={item.color as any} text={item.text} />;
      },
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 150,
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '账号',
      dataIndex: 'accountName',
      key: 'accountName',
      width: 150,
      render: (text: string) => text || '-',
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      width: 120,
      render: (text: string) => text || '-',
    },
    {
      title: '请求ID',
      dataIndex: 'requestId',
      key: 'requestId',
      width: 200,
      ellipsis: true,
      render: (text: string) => text || '-',
    },
  ];

  const alertRuleColumns: ProColumns<AlertRule>[] = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '监控指标',
      dataIndex: 'metric',
      key: 'metric',
      width: 150,
      render: (metric: string) => {
        const metricMap: Record<string, string> = {
          qps: 'QPS',
          latency: '延迟',
          error_rate: '错误率',
          cpu_usage: 'CPU使用率',
          memory_usage: '内存使用率',
        };
        return <Tag>{metricMap[metric] || metric}</Tag>;
      },
    },
    {
      title: '条件',
      dataIndex: 'operator',
      key: 'operator',
      width: 100,
      render: (op: string, record) => {
        const opMap: Record<string, string> = {
          gt: '>',
          lt: '<',
          eq: '=',
          gte: '>=',
          lte: '<=',
        };
        return <span>{opMap[op]} {record.threshold}</span>;
      },
    },
    {
      title: '持续时间',
      dataIndex: 'duration',
      key: 'duration',
      width: 120,
      render: (duration: number) => `${duration} 分钟`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'default'}>
          {status === 'active' ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '通知渠道',
      dataIndex: 'channels',
      key: 'channels',
      width: 200,
      render: (channels: string[]) => (
        <Space size="small" wrap>
          {channels?.map((ch) => (
            <Tag key={ch} size="small">{ch}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      valueType: 'dateTime',
    },
  ];

  return (
    <PageContainer
      title="运维监控"
      subTitle="实时监控系统运行状态，查看错误日志和告警规则"
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane
          tab={
            <span>
              <DashboardOutlined />
              监控大盘
            </span>
          }
          key="dashboard"
        >
          {/* 核心指标 */}
          <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="当前并发"
                  value={concurrencyStats?.total || 0}
                  suffix="个"
                  prefix={<TeamOutlined />}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="当前 QPS"
                  value={trafficData[trafficData.length - 1]?.qps || 0}
                  suffix="req/s"
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="平均延迟"
                  value={trafficData[trafficData.length - 1]?.latency || 0}
                  suffix="ms"
                  valueStyle={{ color: '#722ed1' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="错误率"
                  value={trafficData[trafficData.length - 1]?.errorRate || 0}
                  suffix="%"
                  precision={2}
                  valueStyle={{ color: '#ff4d4f' }}
                />
              </Card>
            </Col>
          </Row>

          {/* 图表区域 */}
          <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
            <Col xs={24} lg={12}>
              <Card>
                <ReactECharts option={qpsOption} style={{ height: 300 }} />
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card>
                <ReactECharts option={latencyOption} style={{ height: 300 }} />
              </Card>
            </Col>
          </Row>

          <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
            <Col xs={24} lg={12}>
              <Card>
                <ReactECharts option={errorRateOption} style={{ height: 300 }} />
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card>
                <ReactECharts option={concurrencyByProviderOption} style={{ height: 300 }} />
              </Card>
            </Col>
          </Row>

          {/* 并发详情 */}
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card title="按模型并发">
                <Table
                  dataSource={concurrencyStats?.byModel || []}
                  rowKey="model"
                  pagination={false}
                  size="small"
                  columns={[
                    { title: '模型', dataIndex: 'model', key: 'model' },
                    { title: '并发数', dataIndex: 'count', key: 'count' },
                  ]}
                />
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card title="按用户并发 TOP10">
                <Table
                  dataSource={concurrencyStats?.byUser?.slice(0, 10) || []}
                  rowKey="userId"
                  pagination={false}
                  size="small"
                  columns={[
                    { title: '用户', dataIndex: 'username', key: 'username' },
                    { title: '并发数', dataIndex: 'count', key: 'count' },
                  ]}
                />
              </Card>
            </Col>
          </Row>
        </TabPane>

        <TabPane
          tab={
            <span>
              <BugOutlined />
              错误日志
            </span>
          }
          key="errors"
        >
          <Alert
            message="错误日志说明"
            description="记录系统运行过程中的错误和警告信息，帮助排查问题。"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />

          <ProTable<ErrorLog>
            headerTitle="错误日志列表"
            rowKey="id"
            search={false}
            toolBarRender={() => [
              <Button key="refresh" icon={<ReloadOutlined />} onClick={fetchErrorLogs}>
                刷新
              </Button>,
            ]}
            dataSource={errorLogs}
            columns={errorLogColumns}
            pagination={{
              showQuickJumper: true,
              showSizeChanger: true,
              defaultPageSize: 20,
            }}
          />
        </TabPane>

        <TabPane
          tab={
            <span>
              <AlertOutlined />
              告警规则
            </span>
          }
          key="alerts"
        >
          <Alert
            message="告警规则说明"
            description="当监控指标满足设定条件并持续指定时间后，系统将发送告警通知到配置的渠道。"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />

          <ProTable<AlertRule>
            headerTitle="告警规则列表"
            rowKey="id"
            search={false}
            toolBarRender={() => [
              <Button key="refresh" icon={<ReloadOutlined />} onClick={fetchAlertRules}>
                刷新
              </Button>,
            ]}
            dataSource={alertRules}
            columns={alertRuleColumns}
            pagination={false}
          />
        </TabPane>
      </Tabs>
    </PageContainer>
  );
};

export default MonitoringPage;
