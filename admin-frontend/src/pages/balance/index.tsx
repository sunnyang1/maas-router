import React, { useState, useEffect, useCallback } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  message,
  Typography,
  Tooltip,
  RefreshOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

interface BalanceRecord {
  id: string;
  accountName: string;
  platform: string;
  balance: number;
  currency: string;
  usedToday: number;
  lastUpdated: string;
  status: 'active' | 'inactive' | 'error';
}

const BalancePage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<BalanceRecord[]>([]);
  const [autoRefreshKey, setAutoRefreshKey] = useState(0);

  const fetchBalances = useCallback(async () => {
    setLoading(true);
    try {
      // TODO: replace with actual API call
      // const res = await getAllBalances();
      // setData(res.data || []);
      setData([]);
    } catch (err) {
      message.error('获取余额数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleRefresh = async () => {
    try {
      // TODO: replace with actual API call
      // await refreshBalance();
      message.success('余额已刷新');
      await fetchBalances();
    } catch (err) {
      message.error('刷新余额失败');
    }
  };

  useEffect(() => {
    fetchBalances();
  }, [fetchBalances]);

  // Auto-refresh every 5 minutes
  useEffect(() => {
    const timer = setInterval(() => {
      fetchBalances();
      setAutoRefreshKey((prev) => prev + 1);
    }, 5 * 60 * 1000);
    return () => clearInterval(timer);
  }, [fetchBalances]);

  const getBalanceColor = (balance: number) => {
    if (balance > 10) return '#52c41a'; // green
    if (balance > 1) return '#faad14';  // yellow
    return '#ff4d4f';                   // red
  };

  const getStatusTag = (status: string) => {
    const statusMap: Record<string, { color: string; text: string }> = {
      active: { color: 'green', text: '正常' },
      inactive: { color: 'default', text: '停用' },
      error: { color: 'red', text: '异常' },
    };
    const info = statusMap[status] || { color: 'default', text: status };
    return <Tag color={info.color}>{info.text}</Tag>;
  };

  const columns: ColumnsType<BalanceRecord> = [
    {
      title: '账号名称',
      dataIndex: 'accountName',
      key: 'accountName',
      ellipsis: true,
    },
    {
      title: '平台',
      dataIndex: 'platform',
      key: 'platform',
      width: 120,
      filters: [
        { text: 'Anthropic', value: 'anthropic' },
        { text: 'OpenAI', value: 'openai' },
        { text: 'Azure', value: 'azure' },
        { text: 'Google', value: 'google' },
      ],
      onFilter: (value, record) => record.platform === value,
    },
    {
      title: '余额',
      dataIndex: 'balance',
      key: 'balance',
      width: 140,
      sorter: (a, b) => a.balance - b.balance,
      render: (balance: number, record) => (
        <Text strong style={{ color: getBalanceColor(balance) }}>
          {record.currency === 'USD' ? '$' : record.currency}
          {balance.toFixed(2)}
        </Text>
      ),
    },
    {
      title: '今日使用',
      dataIndex: 'usedToday',
      key: 'usedToday',
      width: 120,
      sorter: (a, b) => a.usedToday - b.usedToday,
      render: (used: number, record) => (
        <Text>
          {record.currency === 'USD' ? '$' : record.currency}
          {used.toFixed(2)}
        </Text>
      ),
    },
    {
      title: '最后更新',
      dataIndex: 'lastUpdated',
      key: 'lastUpdated',
      width: 180,
      sorter: (a, b) => new Date(a.lastUpdated).getTime() - new Date(b.lastUpdated).getTime(),
      render: (text: string) => (
        <Tooltip title={text}>
          <Space>
            <ClockCircleOutlined />
            <Text type="secondary">{text ? new Date(text).toLocaleString() : '-'}</Text>
          </Space>
        </Tooltip>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      filters: [
        { text: '正常', value: 'active' },
        { text: '停用', value: 'inactive' },
        { text: '异常', value: 'error' },
      ],
      onFilter: (value, record) => record.status === value,
      render: (status: string) => getStatusTag(status),
    },
  ];

  return (
    <PageContainer
      header={{
        title: '账号余额',
        subTitle: '查看所有上游账号的余额和使用情况',
      }}
      extra={[
        <Tooltip title="自动刷新间隔: 5分钟" key="auto-refresh">
          <Tag icon={<ClockCircleOutlined />} color="blue">
            自动刷新中
          </Tag>
        </Tooltip>,
        <Button
          key="refresh"
          type="primary"
          icon={<RefreshOutlined />}
          onClick={handleRefresh}
          loading={loading}
        >
          刷新余额
        </Button>,
      ]}
    >
      <Card>
        <Table<BalanceRecord>
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            defaultPageSize: 20,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 个账号`,
          }}
          scroll={{ x: 800 }}
        />
      </Card>
    </PageContainer>
  );
};

export default BalancePage;
