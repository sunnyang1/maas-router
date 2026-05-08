import React, { useRef, useState } from 'react';
import { PageContainer, ProTable, ProColumns, ActionType } from '@ant-design/pro-components';
import {
  Button,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  Space,
  Popconfirm,
  message,
  Switch,
  InputNumber,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { AccountInfo } from '@/services/api';
import { getAccounts, createAccount, updateAccount, deleteAccount, testAccount, refreshAccountToken } from '@/services/api';

const { Option } = Select;
const { TextArea } = Input;

const ProviderTypeMap: Record<string, { color: string; text: string }> = {
  openai: { color: 'blue', text: 'OpenAI' },
  anthropic: { color: 'purple', text: 'Anthropic' },
  azure: { color: 'cyan', text: 'Azure' },
  google: { color: 'green', text: 'Google' },
  custom: { color: 'default', text: '自定义' },
};

const AccountStatusMap: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
  active: { color: 'success', text: '正常', icon: <CheckCircleOutlined /> },
  inactive: { color: 'default', text: '未激活', icon: <CloseCircleOutlined /> },
  error: { color: 'error', text: '异常', icon: <CloseCircleOutlined /> },
};

const AccountList: React.FC = () => {
  const actionRef = useRef<ActionType>();
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('新增账号');
  const [editingAccount, setEditingAccount] = useState<AccountInfo | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [form] = Form.useForm();

  const columns: ProColumns<AccountInfo>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      fixed: 'left',
    },
    {
      title: '供应商',
      dataIndex: 'provider',
      key: 'provider',
      width: 120,
      valueEnum: {
        openai: { text: 'OpenAI' },
        anthropic: { text: 'Anthropic' },
        azure: { text: 'Azure' },
        google: { text: 'Google' },
        custom: { text: '自定义' },
      },
      render: (_, record) => (
        <Tag color={ProviderTypeMap[record.provider]?.color}>
          {ProviderTypeMap[record.provider]?.text}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (_, record) => {
        const status = AccountStatusMap[record.status];
        return (
          <Tag color={status.color} icon={status.icon}>
            {status.text}
          </Tag>
        );
      },
    },
    {
      title: 'API Endpoint',
      dataIndex: 'apiEndpoint',
      key: 'apiEndpoint',
      width: 250,
      ellipsis: true,
      render: (text: string) => text || '-',
    },
    {
      title: '分组',
      dataIndex: 'groupName',
      key: 'groupName',
      width: 120,
      hideInSearch: true,
      render: (text: string) => text ? <Tag>{text}</Tag> : '-',
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: true,
    },
    {
      title: '权重',
      dataIndex: 'weight',
      key: 'weight',
      width: 80,
    },
    {
      title: '模型',
      dataIndex: 'models',
      key: 'models',
      width: 150,
      hideInSearch: true,
      render: (models: string[]) => (
        <Tooltip title={models?.join(', ')}>
          <Tag>{models?.length || 0} 个</Tag>
        </Tooltip>
      ),
    },
    {
      title: '最后使用',
      dataIndex: 'lastUsedAt',
      key: 'lastUsedAt',
      width: 170,
      hideInSearch: true,
      render: (text: string) => text || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      hideInSearch: true,
      valueType: 'dateTime',
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 250,
      hideInSearch: true,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="text"
            icon={<SyncOutlined spin={testingId === record.id} />}
            onClick={() => handleTest(record.id)}
            loading={testingId === record.id}
          >
            测试
          </Button>
          <Button
            type="text"
            icon={<ReloadOutlined />}
            onClick={() => handleRefreshToken(record.id)}
          >
            刷新
          </Button>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除"
            description="删除后无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const handleAdd = () => {
    setEditingAccount(null);
    setModalTitle('新增账号');
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: AccountInfo) => {
    setEditingAccount(record);
    setModalTitle('编辑账号');
    form.setFieldsValue({
      name: record.name,
      provider: record.provider,
      apiEndpoint: record.apiEndpoint,
      priority: record.priority,
      weight: record.weight,
      status: record.status,
      models: record.models,
      apiKey: record.apiKey,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    try {
      const response = await deleteAccount(id);
      if (response.success) {
        message.success('删除成功');
        actionRef.current?.reload();
      } else {
        message.error(response.message || '删除失败');
      }
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleTest = async (id: string) => {
    setTestingId(id);
    try {
      const response = await testAccount(id);
      if (response.success && response.data.success) {
        message.success(`连接测试成功，延迟: ${response.data.latency}ms`);
      } else {
        message.error(`连接测试失败: ${response.data?.error || '未知错误'}`);
      }
    } catch (error) {
      message.error('连接测试失败');
    } finally {
      setTestingId(null);
    }
  };

  const handleRefreshToken = async (id: string) => {
    try {
      const response = await refreshAccountToken(id);
      if (response.success) {
        message.success('Token 刷新成功');
        actionRef.current?.reload();
      } else {
        message.error(response.message || 'Token 刷新失败');
      }
    } catch (error) {
      message.error('Token 刷新失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const data = {
        ...values,
        apiKey: values.apiKey,
      };

      if (editingAccount) {
        const response = await updateAccount(editingAccount.id, data);
        if (response.success) {
          message.success('更新成功');
          setModalVisible(false);
          actionRef.current?.reload();
        } else {
          message.error(response.message || '更新失败');
        }
      } else {
        const response = await createAccount(data);
        if (response.success) {
          message.success('创建成功');
          setModalVisible(false);
          actionRef.current?.reload();
        } else {
          message.error(response.message || '创建失败');
        }
      }
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  return (
    <PageContainer
      title="账号管理"
      subTitle="管理 LLM 服务账号，配置 API 接入和负载均衡"
    >
      <ProTable<AccountInfo>
        headerTitle="账号列表"
        actionRef={actionRef}
        rowKey="id"
        search={{
          labelWidth: 120,
          defaultCollapsed: false,
        }}
        toolBarRender={() => [
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增账号
          </Button>,
        ]}
        request={async (params) => {
          const { current, pageSize, ...restParams } = params;
          const response = await getAccounts({
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
        scroll={{ x: 1500 }}
      />

      <Modal
        title={modalTitle}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={700}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ priority: 1, weight: 100, status: 'active' }}
        >
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入账号名称' }]}
          >
            <Input placeholder="例如：OpenAI 生产环境" />
          </Form.Item>

          <Form.Item
            name="provider"
            label="供应商"
            rules={[{ required: true, message: '请选择供应商类型' }]}
          >
            <Select placeholder="请选择供应商">
              <Option value="openai">OpenAI</Option>
              <Option value="anthropic">Anthropic</Option>
              <Option value="azure">Azure OpenAI</Option>
              <Option value="google">Google</Option>
              <Option value="custom">自定义</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="apiEndpoint"
            label="API Endpoint"
          >
            <Input placeholder="例如：https://api.openai.com/v1（留空使用默认）" />
          </Form.Item>

          <Form.Item
            name="apiKey"
            label="API Key"
            rules={[{ required: !editingAccount, message: '请输入 API Key' }]}
          >
            <Input.Password placeholder={editingAccount ? '留空表示不修改' : '请输入 API Key'} />
          </Form.Item>

          <Form.Item
            name="models"
            label="支持模型"
          >
            <Select mode="tags" placeholder="输入支持的模型名称">
              <Option value="gpt-4">gpt-4</Option>
              <Option value="gpt-4-turbo">gpt-4-turbo</Option>
              <Option value="gpt-3.5-turbo">gpt-3.5-turbo</Option>
              <Option value="claude-3-opus">claude-3-opus</Option>
              <Option value="claude-3-sonnet">claude-3-sonnet</Option>
              <Option value="gemini-pro">gemini-pro</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: '请输入优先级' }]}
          >
            <InputNumber min={1} max={100} style={{ width: '100%' }} placeholder="数字越小优先级越高" />
          </Form.Item>

          <Form.Item
            name="weight"
            label="权重"
            rules={[{ required: true, message: '请输入权重' }]}
          >
            <InputNumber min={1} max={1000} style={{ width: '100%' }} placeholder="用于负载均衡" />
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            valuePropName="checked"
            getValueFromEvent={(checked) => checked ? 'active' : 'inactive'}
            getValueProps={(value) => ({ checked: value === 'active' })}
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default AccountList;
