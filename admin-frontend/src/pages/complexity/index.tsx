/**
 * 复杂度分析引擎 - 运维管理面板
 *
 * 提供复杂度分析引擎的运维管理功能，包括：
 * - 统计概览（总分析次数、平均复杂度评分、成本节省总额、自动升级率）
 * - 复杂度分布表格
 * - 模型分级配置表格
 * - 质量保障统计
 */
import React, { useEffect, useState } from 'react';
import {
  PageContainer,
  ProCard,
  ProTable,
  ProColumns,
  StatisticCard,
} from '@ant-design/pro-components';
import {
  Card,
  Row,
  Col,
  Statistic,
  Tag,
  Space,
  Spin,
  Alert,
  Table,
  Progress,
  Tooltip,
  Button,
  ReloadOutlined,
} from 'antd';
import {
  ThunderboltOutlined,
  DashboardOutlined,
  DollarOutlined,
  ArrowUpOutlined,
  SafetyCertificateOutlined,
  ExperimentOutlined,
  SettingOutlined,
} from '@ant-design/icons';

// ============== 类型定义 ==============

interface ComplexityStats {
  totalAnalysisCount: number;
  averageComplexityScore: number;
  totalCostSaving: number;
  autoUpgradeRate: number;
}

interface ComplexityDistributionItem {
  key: string;
  level: string;
  requests: number;
  percentage: number;
  avgCostSaving: number;
}

interface ModelTierConfigItem {
  key: string;
  tierName: string;
  models: string[];
  threshold: number;
  costPerToken: number;
  priority: number;
}

interface QualityGuardStats {
  passRate: number;
  samplingRate: number;
  upgradeThreshold: number;
}

// ============== 颜色映射 ==============

const LEVEL_COLOR_MAP: Record<string, string> = {
  simple: 'green',
  medium: 'blue',
  complex: 'orange',
  expert: 'red',
};

const LEVEL_LABEL_MAP: Record<string, string> = {
  simple: '简单',
  medium: '中等',
  complex: '复杂',
  expert: '专家',
};

const PRIORITY_COLOR_MAP: Record<number, string> = {
  1: 'red',
  2: 'volcano',
  3: 'orange',
};

// ============== 主组件 ==============

const ComplexityAnalysisPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<ComplexityStats>({
    totalAnalysisCount: 0,
    averageComplexityScore: 0,
    totalCostSaving: 0,
    autoUpgradeRate: 0,
  });
  const [distribution, setDistribution] = useState<ComplexityDistributionItem[]>([]);
  const [modelTiers, setModelTiers] = useState<ModelTierConfigItem[]>([]);
  const [qualityGuard, setQualityGuard] = useState<QualityGuardStats>({
    passRate: 0,
    samplingRate: 0,
    upgradeThreshold: 0,
  });

  useEffect(() => {
    fetchAllData();
  }, []);

  const fetchAllData = async () => {
    setLoading(true);
    try {
      await Promise.all([
        fetchStats(),
        fetchDistribution(),
        fetchModelTiers(),
        fetchQualityGuard(),
      ]);
    } catch (error) {
      console.error('Failed to fetch complexity data:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const response = await fetch('/api/v1/complexity/admin/stats');
      if (response.ok) {
        const data = await response.json();
        setStats(data);
      }
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  };

  const fetchDistribution = async () => {
    try {
      const response = await fetch('/api/v1/complexity/admin/distribution');
      if (response.ok) {
        const data = await response.json();
        setDistribution(
          (data || []).map((item: any, index: number) => ({
            key: item.level || String(index),
            level: item.level,
            requests: item.requests || item.count || 0,
            percentage: item.percentage || 0,
            avgCostSaving: item.avgCostSaving || item.avg_cost_saving || 0,
          }))
        );
      }
    } catch (error) {
      console.error('Failed to fetch distribution:', error);
    }
  };

  const fetchModelTiers = async () => {
    try {
      const response = await fetch('/api/v1/complexity/admin/model-tiers');
      if (response.ok) {
        const data = await response.json();
        setModelTiers(
          (data || []).map((item: any, index: number) => ({
            key: item.tierName || item.tier_name || String(index),
            tierName: item.tierName || item.tier_name || '-',
            models: item.models || [],
            threshold: item.threshold || 0,
            costPerToken: item.costPerToken || item.cost_per_token || 0,
            priority: item.priority || 0,
          }))
        );
      }
    } catch (error) {
      console.error('Failed to fetch model tiers:', error);
    }
  };

  const fetchQualityGuard = async () => {
    try {
      const response = await fetch('/api/v1/complexity/admin/quality-guard');
      if (response.ok) {
        const data = await response.json();
        setQualityGuard(data);
      }
    } catch (error) {
      console.error('Failed to fetch quality guard stats:', error);
    }
  };

  // ============== 表格列定义 ==============

  const distributionColumns: ProColumns<ComplexityDistributionItem>[] = [
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 120,
      render: (level: string) => (
        <Tag color={LEVEL_COLOR_MAP[level] || 'default'}>
          {LEVEL_LABEL_MAP[level] || level}
        </Tag>
      ),
    },
    {
      title: '请求数',
      dataIndex: 'requests',
      key: 'requests',
      width: 150,
      sorter: (a, b) => a.requests - b.requests,
      render: (val: number) => val?.toLocaleString() || '0',
    },
    {
      title: '占比',
      dataIndex: 'percentage',
      key: 'percentage',
      width: 200,
      sorter: (a, b) => a.percentage - b.percentage,
      render: (val: number) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Progress
            percent={val || 0}
            size="small"
            strokeColor={val > 40 ? '#ff4d4f' : val > 25 ? '#faad14' : '#52c41a'}
            style={{ flex: 1, marginBottom: 0 }}
          />
          <span>{val?.toFixed(1)}%</span>
        </div>
      ),
    },
    {
      title: '平均成本节省',
      dataIndex: 'avgCostSaving',
      key: 'avgCostSaving',
      width: 150,
      sorter: (a, b) => a.avgCostSaving - b.avgCostSaving,
      render: (val: number) => (
        <span style={{ color: val > 30 ? '#52c41a' : val > 15 ? '#faad14' : '#ff4d4f' }}>
          {val?.toFixed(1)}%
        </span>
      ),
    },
  ];

  const modelTierColumns: ProColumns<ModelTierConfigItem>[] = [
    {
      title: '层级名称',
      dataIndex: 'tierName',
      key: 'tierName',
      width: 150,
      render: (name: string) => (
        <Space>
          <SettingOutlined />
          <span style={{ fontWeight: 500 }}>{name}</span>
        </Space>
      ),
    },
    {
      title: '包含模型',
      dataIndex: 'models',
      key: 'models',
      width: 300,
      render: (models: string[]) => (
        <Space size={[4, 4]} wrap>
          {(models || []).map((model) => (
            <Tag key={model}>{model}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '复杂度阈值',
      dataIndex: 'threshold',
      key: 'threshold',
      width: 130,
      sorter: (a, b) => a.threshold - b.threshold,
      render: (val: number) => (
        <Tooltip title="复杂度评分达到此阈值时使用该层级模型">
          <Tag color="blue">{val?.toFixed(2)}</Tag>
        </Tooltip>
      ),
    },
    {
      title: '每 Token 成本',
      dataIndex: 'costPerToken',
      key: 'costPerToken',
      width: 140,
      sorter: (a, b) => a.costPerToken - b.costPerToken,
      render: (val: number) => (
        <span style={{ fontFamily: 'monospace' }}>
          ${val?.toExponential(2) || '0'}
        </span>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      sorter: (a, b) => a.priority - b.priority,
      render: (priority: number) => (
        <Tag color={PRIORITY_COLOR_MAP[priority] || 'default'}>
          P{priority}
        </Tag>
      ),
    },
  ];

  return (
    <PageContainer
      title="复杂度分析引擎"
      subTitle="智能推理资源优化引擎的运维管理面板"
      extra={[
        <Button
          key="refresh"
          icon={<ReloadOutlined />}
          onClick={fetchAllData}
          loading={loading}
        >
          刷新数据
        </Button>,
      ]}
    >
      <Spin spinning={loading}>
        {/* 统计卡片行 */}
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} lg={6}>
            <ProCard hoverable>
              <Statistic
                title={
                  <Space>
                    <ThunderboltOutlined />
                    总分析次数
                  </Space>
                }
                value={stats.totalAnalysisCount}
                precision={0}
                valueStyle={{ color: '#1890ff' }}
                suffix="次"
              />
            </ProCard>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <ProCard hoverable>
              <Statistic
                title={
                  <Space>
                    <DashboardOutlined />
                    平均复杂度评分
                  </Space>
                }
                value={stats.averageComplexityScore}
                precision={2}
                valueStyle={{ color: '#722ed1' }}
                suffix="/ 1.0"
              />
            </ProCard>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <ProCard hoverable>
              <Statistic
                title={
                  <Space>
                    <DollarOutlined />
                    成本节省总额
                  </Space>
                }
                value={stats.totalCostSaving}
                precision={2}
                prefix="$"
                valueStyle={{ color: '#52c41a' }}
                suffix={
                  <span style={{ fontSize: 14 }}>
                    <ArrowUpOutlined style={{ color: '#52c41a', marginLeft: 4 }} />
                  </span>
                }
              />
            </ProCard>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <ProCard hoverable>
              <Statistic
                title={
                  <Space>
                    <ArrowUpOutlined />
                    自动升级率
                  </Space>
                }
                value={stats.autoUpgradeRate}
                precision={1}
                suffix="%"
                valueStyle={{
                  color: stats.autoUpgradeRate < 10 ? '#52c41a' : stats.autoUpgradeRate < 25 ? '#faad14' : '#ff4d4f',
                }}
              />
            </ProCard>
          </Col>
        </Row>

        {/* 复杂度分布表格 */}
        <ProCard
          title={
            <Space>
              <ExperimentOutlined />
              复杂度分布
            </Space>
          }
          style={{ marginBottom: 24 }}
          headerBordered
        >
          <ProTable<ComplexityDistributionItem>
            headerTitle="请求复杂度分级统计"
            rowKey="key"
            search={false}
            toolBarRender={false}
            dataSource={distribution}
            columns={distributionColumns}
            pagination={false}
            size="small"
          />
        </ProCard>

        {/* 模型分级配置表格 */}
        <ProCard
          title={
            <Space>
              <SettingOutlined />
              模型分级配置
            </Space>
          }
          style={{ marginBottom: 24 }}
          headerBordered
        >
          <Alert
            message="模型分级说明"
            description="根据请求复杂度评分自动选择合适的模型层级，在保证质量的同时优化成本。复杂度阈值表示评分达到该值时推荐使用对应层级的模型。"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <ProTable<ModelTierConfigItem>
            headerTitle="模型层级配置列表"
            rowKey="key"
            search={false}
            toolBarRender={false}
            dataSource={modelTiers}
            columns={modelTierColumns}
            pagination={false}
            size="small"
            scroll={{ x: 820 }}
          />
        </ProCard>

        {/* 质量保障统计 */}
        <ProCard
          title={
            <Space>
              <SafetyCertificateOutlined />
              质量保障统计
            </Space>
          }
          headerBordered
        >
          <Row gutter={[24, 24]}>
            <Col xs={24} md={8}>
              <StatisticCard
                statistic={{
                  title: '质检通过率',
                  value: qualityGuard.passRate,
                  precision: 1,
                  suffix: '%',
                  valueStyle: {
                    color: qualityGuard.passRate > 95 ? '#52c41a' : qualityGuard.passRate > 85 ? '#faad14' : '#ff4d4f',
                  },
                  description: (
                    <Space direction="vertical" size={4}>
                      <span style={{ color: '#999', fontSize: 12 }}>
                        自动质量检测结果中通过的比例
                      </span>
                      <Progress
                        percent={qualityGuard.passRate || 0}
                        strokeColor={
                          qualityGuard.passRate > 95
                            ? '#52c41a'
                            : qualityGuard.passRate > 85
                            ? '#faad14'
                            : '#ff4d4f'
                        }
                        size="small"
                      />
                    </Space>
                  ),
                }}
              />
            </Col>
            <Col xs={24} md={8}>
              <StatisticCard
                statistic={{
                  title: '采样率',
                  value: qualityGuard.samplingRate,
                  precision:1,
                  suffix: '%',
                  valueStyle: { color: '#1890ff' },
                  description: (
                    <Space direction="vertical" size={4}>
                      <span style={{ color: '#999', fontSize: 12 }}>
                        自动升级请求的采样检测比例
                      </span>
                      <Progress
                        percent={qualityGuard.samplingRate || 0}
                        strokeColor="#1890ff"
                        size="small"
                      />
                    </Space>
                  ),
                }}
              />
            </Col>
            <Col xs={24} md={8}>
              <StatisticCard
                statistic={{
                  title: '升级阈值',
                  value: qualityGuard.upgradeThreshold,
                  precision: 2,
                  suffix: '',
                  valueStyle: { color: '#722ed1' },
                  description: (
                    <Space direction="vertical" size={4}>
                      <span style={{ color: '#999', fontSize: 12 }}>
                        复杂度评分超过此值时自动升级模型层级
                      </span>
                      <Progress
                        percent={(qualityGuard.upgradeThreshold || 0) * 100}
                        strokeColor="#722ed1"
                        size="small"
                        format={(percent) => `${(percent || 0) / 100}`}
                      />
                    </Space>
                  ),
                }}
              />
            </Col>
          </Row>
        </ProCard>
      </Spin>
    </PageContainer>
  );
};

export default ComplexityAnalysisPage;
