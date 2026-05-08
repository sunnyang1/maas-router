import React, { useState, useCallback } from 'react';
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
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  QuestionCircleOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

interface TestResult {
  id: string;
  accountName: string;
  platform: string;
  status: 'healthy' | 'unhealthy' | 'unknown' | 'testing';
  latency: number;
  lastTest: string;
  error: string;
}

const ChannelTestPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [testingAll, setTestingAll] = useState(false);
  const [testingIds, setTestingIds] = useState<Set<string>>(new Set());
  const [data, setData] = useState<TestResult[]>([]);

  const fetchResults = useCallback(async () => {
    setLoading(true);
    try {
      // TODO: replace with actual API call
      // const res = await getTestResults();
      // setData(res.data || []);
      setData([]);
    } catch (err) {
      message.error('获取测试结果失败');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleTestOne = async (record: TestResult) => {
    setTestingIds((prev) => new Set(prev).add(record.id));
    // Optimistically update status
    setData((prev) =>
      prev.map((item) =>
        item.id === record.id ? { ...item, status: 'testing' as const } : item,
      ),
    );
    try {
      // TODO: replace with actual API call
      // const res = await testAccount(record.id);
      // Update the record with results
      setData((prev) =>
        prev.map((item) =>
          item.id === record.id
            ? {
                ...item,
                status: 'healthy' as const,
                latency: Math.floor(Math.random() * 500) + 50,
                lastTest: new Date().toISOString(),
                error: '',
              }
            : item,
        ),
      );
      message.success(`${record.accountName} 测试通过`);
    } catch (err) {
      setData((prev) =>
        prev.map((item) =>
          item.id === record.id
            ? {
                ...item,
                status: 'unhealthy' as const,
                lastTest: new Date().toISOString(),
                error: '连接失败',
              }
            : item,
        ),
      );
      message.error(`${record.accountName} 测试失败`);
    } finally {
      setTestingIds((prev) => {
        const next = new Set(prev);
        next.delete(record.id);
        return next;
      });
    }
  };

  const handleTestAll = async () => {
    setTestingAll(true);
    try {
      // TODO: replace with actual API call
      // await testAllAccounts();
      // Simulate: mark all as testing, then resolve
      setData((prev) =>
        prev.map((item) => ({ ...item, status: 'testing' as const })),
      );

      // Simulate sequential test completion
      for (let i = 0; i < data.length; i++) {
        await new Promise((resolve) => setTimeout(resolve, 300));
        setData((prev) =>
          prev.map((item, idx) =>
            idx === i
              ? {
                  ...item,
                  status: (Math.random() > 0.2 ? 'healthy' : 'unhealthy') as 'healthy' | 'unhealthy',
                  latency: Math.floor(Math.random() * 500) + 50,
                  lastTest: new Date().toISOString(),
                  error: Math.random() > 0.2 ? '' : '连接超时',
                }
              : item,
          ),
        );
      }
      message.success('全部测试完成');
    } catch (err) {
      message.error('批量测试失败');
    } finally {
      setTestingAll(false);
    }
  };

  const getStatusTag = (status: string) => {
    switch (status) {
      case 'healthy':
        return (
          <Tag icon={<CheckCircleOutlined />} color="success">
            健康
          </Tag>
        );
      case 'unhealthy':
        return (
          <Tag icon={<CloseCircleOutlined />} color="error">
            异常
          </Tag>
        );
      case 'testing':
        return (
          <Tag color="processing">
            <ReloadOutlined spin /> 测试中
          </Tag>
        );
      default:
        return (
          <Tag icon={<QuestionCircleOutlined />} color="default">
            未知
          </Tag>
        );
    }
  };

  const columns: ColumnsType<TestResult> = [
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
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      filters: [
        { text: '健康', value: 'healthy' },
        { text: '异常', value: 'unhealthy' },
        { text: '未知', value: 'unknown' },
      ],
      onFilter: (value, record) => record.status === value,
      render: (status: string) => getStatusTag(status),
    },
    {
      title: '延迟',
      dataIndex: 'latency',
      key: 'latency',
      width: 120,
      sorter: (a, b) => a.latency - b.latency,
      render: (latency: number, record) => {
        if (record.status === 'testing' || record.status === 'unknown') return '-';
        const color = latency < 200 ? '#52c41a' : latency < 500 ? '#faad14' : '#ff4d4f';
        return <Text style={{ color }}>{latency}ms</Text>;
      },
    },
    {
      title: '最后测试',
      dataIndex: 'lastTest',
      key: 'lastTest',
      width: 180,
      render: (text: string) =>
        text ? (
          <Tooltip title={text}>
            <Space>
              <ClockCircleOutlined />
              <Text type="secondary">{new Date(text).toLocaleString()}</Text>
            </Space>
          </Tooltip>
        ) : (
          <Text type="secondary">未测试</Text>
        ),
    },
    {
      title: '错误信息',
      dataIndex: 'error',
      key: 'error',
      width: 200,
      ellipsis: true,
      render: (error: string) =>
        error ? (
          <Tooltip title={error}>
            <Text type="danger">{error}</Text>
          </Tooltip>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<ThunderboltOutlined />}
          loading={testingIds.has(record.id)}
          onClick={() => handleTestOne(record)}
          disabled={record.status === 'testing'}
        >
          测试
        </Button>
      ),
    },
  ];

  return (
    <PageContainer
      header={{
        title: '渠道测试',
        subTitle: '测试上游账号的连接健康状态',
      }}
      extra={[
        <Button
          key="refresh"
          icon={<ReloadOutlined />}
          onClick={fetchResults}
          loading={loading}
        >
          刷新结果
        </Button>,
        <Button
          key="test-all"
          type="primary"
          icon={<ThunderboltOutlined />}
          onClick={handleTestAll}
          loading={testingAll}
        >
          测试全部
        </Button>,
      ]}
    >
      <Card>
        <Table<TestResult>
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            defaultPageSize: 20,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 个账号`,
          }}
          scroll={{ x: 900 }}
          rowClassName={(record) => {
            switch (record.status) {
              case 'healthy':
                return 'row-healthy';
              case 'unhealthy':
                return 'row-unhealthy';
              default:
                return '';
            }
          }}
        />
      </Card>

      <style jsx global>{`
        .row-healthy {
          background-color: #f6ffed;
        }
        .row-unhealthy {
          background-color: #fff2f0;
        }
      `}</style>
    </PageContainer>
  );
};

export default ChannelTestPage;
