import React, { useEffect, useState } from 'react';
import { PageContainer, ProCard } from '@ant-design/pro-components';
import {
  Descriptions,
  Tag,
  Space,
  Button,
  Tabs,
  Table,
  Timeline,
  Statistic,
  Row,
  Col,
  Card,
  Progress,
  message,
  Modal,
  Form,
  Input,
  InputNumber,
  Radio,
} from 'antd';
import {
  ArrowLeftOutlined,
  EditOutlined,
  LockOutlined,
  UnlockOutlined,
  KeyOutlined,
  HistoryOutlined,
  BarChartOutlined,
  DollarOutlined,
} from '@ant-design/icons';
import { history, useParams } from '@umijs/max';
import type { UserInfo, ApiKeyInfo, BillingRecord } from '@/services/api';
import { getUser, updateUser, adjustUserBalance, getApiKeys, getBillingRecords } from '@/services/api';
import dayjs from 'dayjs';

const { TabPane } = Tabs;

const UserStatusMap: Record<string, { color: string; text: string }> = {
  active: { color: 'success', text: '正常' },
  inactive: { color: 'default', text: '未激活' },
  banned: { color: 'error', text: '已禁用' },
};

const UserRoleMap: Record<string, { color: string; text: string }> = {
  admin: { color: 'red', text: '管理员' },
  user: { color: 'blue', text: '普通用户' },
  viewer: { color: 'green', text: '访客' },
};

const UserDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [apiKeys, setApiKeys] = useState<ApiKeyInfo[]>([]);
  const [billingRecords, setBillingRecords] = useState<BillingRecord[]>([]);
  const [balanceModalVisible, setBalanceModalVisible] = useState(false);
  const [balanceForm] = Form.useForm();

  useEffect(() => {
    if (id) {
      fetchUserDetail();
      fetchApiKeys();
      fetchBillingRecords();
    }
  }, [id]);

  const fetchUserDetail = async () => {
    try {
      const response = await getUser(id!);
      if (response.success) {
        setUser(response.data);
      }
    } catch (error) {
      message.error('获取用户详情失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchApiKeys = async () => {
    try {
      const response = await getApiKeys({ userId: id, current: 1, pageSize: 100 });
      if (response.success) {
        setApiKeys(response.data.list);
      }
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
    }
  };

  const fetchBillingRecords = async () => {
    try {
      const response = await getBillingRecords({ userId: id, current: 1, pageSize: 10 });
      if (response.success) {
        setBillingRecords(response.data.list);
      }
    } catch (error) {
      console.error('Failed to fetch billing records:', error);
    }
  };

  const handleStatusChange = async (status: string) => {
    try {
      const response = await updateUser(id!, { status: status as UserInfo['status'] });
      if (response.success) {
        message.success(status === 'active' ? '启用成功' : '禁用成功');
        fetchUserDetail();
      } else {
        message.error(response.message || '操作失败');
      }
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleBalanceAdjust = async () => {
    try {
      const values = await balanceForm.validateFields();
      const response = await adjustUserBalance(id!, {
        amount: values.amount,
        type: values.type,
        description: values.description,
      });
      if (response.success) {
        message.success('余额调整成功');
        setBalanceModalVisible(false);
        balanceForm.resetFields();
        fetchUserDetail();
        fetchBillingRecords();
      } else {
        message.error(response.message || '余额调整失败');
      }
    } catch (error) {
      console.error('Balance adjust failed:', error);
    }
  };

  const apiKeyColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Key前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      render: (text: string) => `${text}****`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusMap: Record<string, { color: string; text: string }> = {
          active: { color: 'success', text: '正常' },
          inactive: { color: 'default', text: '未激活' },
          revoked: { color: 'error', text: '已撤销' },
        };
        return <Tag color={statusMap[status]?.color}>{statusMap[status]?.text}</Tag>;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '最后使用',
      dataIndex: 'lastUsedAt',
      key: 'lastUsedAt',
      render: (text: string) => text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
  ];

  const billingColumns = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => {
        const typeMap: Record<string, { color: string; text: string }> = {
          charge: { color: 'green', text: '充值' },
          consumption: { color: 'red', text: '消费' },
          refund: { color: 'blue', text: '退款' },
        };
        return <Tag color={typeMap[type]?.color}>{typeMap[type]?.text}</Tag>;
      },
    },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      render: (amount: number) => (
        <span style={{ color: amount > 0 ? '#52c41a' : '#ff4d4f' }}>
          {amount > 0 ? '+' : ''}{amount.toFixed(2)}
        </span>
      ),
    },
    {
      title: '余额',
      dataIndex: 'balance',
      key: 'balance',
      render: (balance: number) => `¥${balance.toFixed(2)}`,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  if (!user) {
    return null;
  }

  const quotaPercent = user.quotaLimit && user.quotaLimit > 0
    ? Math.round(((user.quotaUsed || 0) / user.quotaLimit) * 100)
    : 0;

  return (
    <PageContainer
      title="用户详情"
      subTitle={user.username}
      loading={loading}
      onBack={() => history.push('/users')}
      extra={[
        <Button
          key="balance"
          icon={<DollarOutlined />}
          onClick={() => setBalanceModalVisible(true)}
        >
          调整余额
        </Button>,
        <Button
          key="edit"
          icon={<EditOutlined />}
          onClick={() => message.info('编辑功能待实现')}
        >
          编辑
        </Button>,
        user.status === 'active' ? (
          <Button
            key="disable"
            danger
            icon={<LockOutlined />}
            onClick={() => handleStatusChange('banned')}
          >
            禁用
          </Button>
        ) : (
          <Button
            key="enable"
            type="primary"
            icon={<UnlockOutlined />}
            onClick={() => handleStatusChange('active')}
          >
            启用
          </Button>
        ),
      ]}
    >
      <ProCard gutter={[16, 16]}>
        <Row gutter={[16, 16]}>
          <Col span={24}>
            <Descriptions title="基本信息" bordered column={{ xs: 1, sm: 2, md: 3, lg: 4 }}>
              <Descriptions.Item label="用户ID">{user.id}</Descriptions.Item>
              <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
              <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
              <Descriptions.Item label="手机号">{user.phone || '-'}</Descriptions.Item>
              <Descriptions.Item label="角色">
                <Tag color={UserRoleMap[user.role]?.color}>
                  {UserRoleMap[user.role]?.text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={UserStatusMap[user.status]?.color}>
                  {UserStatusMap[user.status]?.text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {dayjs(user.createdAt).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="更新时间">
                {dayjs(user.updatedAt).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="最后登录">
                {user.lastLoginAt ? dayjs(user.lastLoginAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
            </Descriptions>
          </Col>
        </Row>

        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="账户余额"
                value={user.balance || 0}
                prefix="¥"
                precision={2}
                valueStyle={{ color: '#52c41a' }}
              />
              <Button 
                type="link" 
                onClick={() => setBalanceModalVisible(true)}
                style={{ padding: 0, marginTop: 8 }}
              >
                调整余额
              </Button>
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="配额使用"
                value={quotaPercent}
                suffix="%"
                precision={1}
              />
              <Progress
                percent={quotaPercent}
                status={quotaPercent > 80 ? 'exception' : 'active'}
                style={{ marginTop: 8 }}
              />
              <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
                {(user.quotaUsed || 0).toLocaleString()} / {(user.quotaLimit || 0).toLocaleString()}
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="API Keys"
                value={apiKeys.length}
                prefix={<KeyOutlined />}
              />
              <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
                活跃: {apiKeys.filter(k => k.status === 'active').length}
              </div>
            </Card>
          </Col>
        </Row>
      </ProCard>

      <ProCard style={{ marginTop: 16 }}>
        <Tabs defaultActiveKey="apikeys">
          <TabPane
            tab={
              <span>
                <KeyOutlined />
                API Keys ({apiKeys.length})
              </span>
            }
            key="apikeys"
          >
            <Table
              columns={apiKeyColumns}
              dataSource={apiKeys}
              rowKey="id"
              pagination={false}
            />
          </TabPane>
          <TabPane
            tab={
              <span>
                <BarChartOutlined />
                消费记录
              </span>
            }
            key="billing"
          >
            <Table
              columns={billingColumns}
              dataSource={billingRecords}
              rowKey="id"
              pagination={false}
            />
          </TabPane>
          <TabPane
            tab={
              <span>
                <HistoryOutlined />
                操作日志
              </span>
            }
            key="logs"
          >
            <Timeline
              items={[
                {
                  color: 'green',
                  children: `用户创建 - ${dayjs(user.createdAt).format('YYYY-MM-DD HH:mm:ss')}`,
                },
                {
                  color: 'blue',
                  children: `信息更新 - ${dayjs(user.updatedAt).format('YYYY-MM-DD HH:mm:ss')}`,
                },
                user.lastLoginAt && {
                  color: 'gray',
                  children: `最后登录 - ${dayjs(user.lastLoginAt).format('YYYY-MM-DD HH:mm:ss')}`,
                },
              ].filter(Boolean)}
            />
          </TabPane>
        </Tabs>
      </ProCard>

      {/* 余额调整弹窗 */}
      <Modal
        title="调整余额"
        open={balanceModalVisible}
        onOk={handleBalanceAdjust}
        onCancel={() => {
          setBalanceModalVisible(false);
          balanceForm.resetFields();
        }}
        width={500}
        destroyOnClose
      >
        <div style={{ marginBottom: 16 }}>
          <span>当前余额: </span>
          <span style={{ fontSize: 18, fontWeight: 'bold', color: '#52c41a' }}>
            ¥{(user.balance || 0).toFixed(2)}
          </span>
        </div>
        <Form
          form={balanceForm}
          layout="vertical"
          initialValues={{ type: 'add' }}
        >
          <Form.Item
            name="type"
            label="操作类型"
            rules={[{ required: true, message: '请选择操作类型' }]}
          >
            <Radio.Group>
              <Radio value="add">增加余额</Radio>
              <Radio value="subtract">扣减余额</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            name="amount"
            label="金额"
            rules={[
              { required: true, message: '请输入金额' },
              { type: 'number', min: 0.01, message: '金额必须大于0' },
            ]}
          >
            <InputNumber
              style={{ width: '100%' }}
              precision={2}
              min={0.01}
              prefix="¥"
              placeholder="请输入金额"
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="备注"
          >
            <Input.TextArea
              rows={3}
              placeholder="请输入备注信息（可选）"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default UserDetail;
