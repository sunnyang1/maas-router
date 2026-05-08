import React, { useRef, useState } from 'react';
import { PageContainer, ProTable, ProColumns, ActionType, ProCard } from '@ant-design/pro-components';
import {
  Button,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  Space,
  message,
  Card,
  Statistic,
  Row,
  Col,
  DatePicker,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  DollarOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons';
import type { BillingRecord, BillingStats } from '@/services/api';
import { getBillingRecords, getBillingStats, recharge } from '@/services/api';
import dayjs from 'dayjs';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';

const { Option } = Select;
const { RangePicker } = DatePicker;

const BillingTypeMap: Record<string, { color: string; text: string }> = {
  charge: { color: 'green', text: '充值' },
  consumption: { color: 'red', text: '消费' },
  refund: { color: 'blue', text: '退款' },
};

const BillingList: React.FC = () => {
  const actionRef = useRef<ActionType>();
  const [modalVisible, setModalVisible] = useState(false);
  const [stats, setStats] = useState<BillingStats | null>(null);
  const [form] = Form.useForm();

  React.useEffect(() => {
    fetchStats();
  }, []);

  const fetchStats = async () => {
    try {
      const response = await getBillingStats();
      if (response.success) {
        setStats(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch billing stats:', error);
    }
  };

  const columns: ProColumns<BillingRecord>[] = [
    {
      title: '记录ID',
      dataIndex: 'id',
      key: 'id',
      width: 220,
      copyable: true,
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      width: 150,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      valueEnum: {
        charge: { text: '充值' },
        consumption: { text: '消费' },
        refund: { text: '退款' },
      },
      render: (_, record) => (
        <Tag color={BillingTypeMap[record.type]?.color}>
          {BillingTypeMap[record.type]?.text}
        </Tag>
      ),
    },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      sorter: true,
      render: (amount: number) => (
        <span style={{ color: amount > 0 ? '#52c41a' : '#ff4d4f', fontWeight: 500 }}>
          {amount > 0 ? '+' : ''}{amount.toFixed(2)}
        </span>
      ),
    },
    {
      title: '余额',
      dataIndex: 'balance',
      key: 'balance',
      width: 120,
      render: (balance: number) => balance.toFixed(2),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 250,
      ellipsis: true,
    },
    {
      title: '关联ID',
      dataIndex: 'relatedId',
      key: 'relatedId',
      width: 200,
      ellipsis: true,
      render: (text) => text || '-',
    },
    {
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      sorter: true,
      valueType: 'dateTime',
    },
  ];

  const handleRecharge = () => {
    form.resetFields();
    setModalVisible(true);
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      await recharge(values.userId, values.amount, values.description);
      message.success('充值成功');
      setModalVisible(false);
      actionRef.current?.reload();
      fetchStats();
    } catch (error) {
      message.error('充值失败');
    }
  };

  // 消费趋势图表配置
  const consumptionTrendOption: EChartsOption = {
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      data: ['充值', '消费'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: '充值',
        type: 'line',
        data: [120, 132, 101, 134, 90, 230, 210],
        itemStyle: { color: '#52c41a' },
      },
      {
        name: '消费',
        type: 'line',
        data: [220, 182, 191, 234, 290, 330, 310],
        itemStyle: { color: '#ff4d4f' },
      },
    ],
  };

  return (
    <PageContainer
      title="计费管理"
      subTitle="管理用户充值、消费记录和账单统计"
    >
      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总收入"
              value={stats?.totalRevenue || 0}
              prefix="¥"
              precision={2}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总消费"
              value={stats?.totalConsumption || 0}
              prefix="¥"
              precision={2}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃用户"
              value={stats?.activeUsers || 0}
              suffix="人"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="今日收入"
              value={stats?.todayRevenue || 0}
              prefix="¥"
              precision={2}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 消费趋势图表 */}
      <Card title="消费趋势" style={{ marginBottom: 24 }}>
        <ReactECharts option={consumptionTrendOption} style={{ height: 300 }} />
      </Card>

      <ProTable<BillingRecord>
        headerTitle="账单记录"
        actionRef={actionRef}
        rowKey="id"
        search={{
          labelWidth: 120,
          defaultCollapsed: false,
        }}
        toolBarRender={() => [
          <Button key="recharge" type="primary" icon={<PlusOutlined />} onClick={handleRecharge}>
            充值
          </Button>,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
        request={async (params) => {
          const { current, pageSize, ...restParams } = params;
          const response = await getBillingRecords({
            current,
            pageSize,
            ...restParams,
          });
          if (response.success) {
            return {
              data: response.data.list,
              total: response.data.total,
              success: true,
            };
          }
          return {
            data: [],
            total: 0,
            success: false,
          };
        }}
        columns={columns}
        pagination={{
          showQuickJumper: true,
          showSizeChanger: true,
          defaultPageSize: 10,
          pageSizeOptions: ['10', '20', '50', '100'],
        }}
        scroll={{ x: 1200 }}
      />

      <Modal
        title="用户充值"
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={500}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
        >
          <Form.Item
            name="userId"
            label="用户"
            rules={[{ required: true, message: '请选择用户' }]}
          >
            <Select placeholder="请选择用户">
              <Option value="user1">user1</Option>
              <Option value="user2">user2</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="amount"
            label="充值金额"
            rules={[
              { required: true, message: '请输入充值金额' },
              { type: 'number', min: 0.01, message: '金额必须大于0' },
            ]}
          >
            <Input type="number" prefix="¥" placeholder="请输入金额" />
          </Form.Item>

          <Form.Item
            name="description"
            label="备注"
          >
            <Input placeholder="可选备注信息" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default BillingList;
