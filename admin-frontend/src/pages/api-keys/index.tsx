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
  Tooltip,
  Descriptions,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  CopyOutlined,
  DeleteOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
} from '@ant-design/icons';
import type { ApiKeyInfo } from '@/services/api';
import { getApiKeys, createApiKey, revokeApiKey } from '@/services/api';
import dayjs from 'dayjs';

const { Option } = Select;
const { TextArea } = Input;

const ApiKeyStatusMap: Record<string, { color: string; text: string }> = {
  active: { color: 'success', text: '正常' },
  inactive: { color: 'default', text: '未激活' },
  revoked: { color: 'error', text: '已撤销' },
};

const ApiKeyList: React.FC = () => {
  const actionRef = useRef<ActionType>();
  const [modalVisible, setModalVisible] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showKey, setShowKey] = useState<Record<string, boolean>>({});
  const [form] = Form.useForm();

  const columns: ProColumns<ApiKeyInfo>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: 'API Key',
      dataIndex: 'key',
      key: 'key',
      width: 300,
      render: (_, record) => {
        const isVisible = showKey[record.id];
        const displayKey = isVisible ? record.key : `${record.keyPrefix}****`;
        return (
          <Space>
            <code style={{ 
              background: '#f5f5f5', 
              padding: '4px 8px', 
              borderRadius: 4,
              fontSize: 12,
            }}>
              {displayKey}
            </code>
            <Button
              type="text"
              size="small"
              icon={isVisible ? <EyeInvisibleOutlined /> : <EyeOutlined />}
              onClick={() => setShowKey(prev => ({ ...prev, [record.id]: !isVisible }))}
            />
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined />}
              onClick={() => {
                navigator.clipboard.writeText(record.key);
                message.success('已复制到剪贴板');
              }}
            />
          </Space>
        );
      },
    },
    {
      title: '用户',
      dataIndex: 'userId',
      key: 'userId',
      width: 150,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      valueEnum: {
        active: { text: '正常', status: 'Success' },
        inactive: { text: '未激活', status: 'Default' },
        revoked: { text: '已撤销', status: 'Error' },
      },
      render: (_, record) => (
        <Tag color={ApiKeyStatusMap[record.status]?.color}>
          {ApiKeyStatusMap[record.status]?.text}
        </Tag>
      ),
    },
    {
      title: '速率限制',
      dataIndex: 'rateLimit',
      key: 'rateLimit',
      width: 120,
      hideInSearch: true,
      render: (limit) => limit ? `${limit} req/min` : '无限制',
    },
    {
      title: '权限',
      dataIndex: 'permissions',
      key: 'permissions',
      width: 200,
      hideInSearch: true,
      render: (permissions: string[]) => (
        <Space size="small" wrap>
          {permissions?.map((perm) => (
            <Tag key={perm} size="small">{perm}</Tag>
          ))}
        </Space>
      ),
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
      title: '过期时间',
      dataIndex: 'expiresAt',
      key: 'expiresAt',
      width: 170,
      hideInSearch: true,
      render: (text) => text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '永不过期',
    },
    {
      title: '最后使用',
      dataIndex: 'lastUsedAt',
      key: 'lastUsedAt',
      width: 170,
      hideInSearch: true,
      render: (text) => text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 150,
      hideInSearch: true,
      render: (_, record) => (
        <Space size="small">
          {record.status === 'active' && (
            <Popconfirm
              title="确认撤销"
              description="撤销后该 API Key 将立即失效，是否继续？"
              onConfirm={() => handleRevoke(record.id)}
            >
              <Button type="text" danger icon={<DeleteOutlined />}>
                撤销
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const handleCreate = () => {
    setCreatedKey(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleRevoke = async (id: string) => {
    try {
      await revokeApiKey(id);
      message.success('API Key 已撤销');
      actionRef.current?.reload();
    } catch (error) {
      message.error('撤销失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const response = await createApiKey(values);
      if (response.success) {
        setCreatedKey(response.data.key);
        message.success('API Key 创建成功');
        actionRef.current?.reload();
      }
    } catch (error) {
      console.error('Failed to create API key:', error);
    }
  };

  const handleCopyKey = () => {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      message.success('已复制到剪贴板');
    }
  };

  return (
    <PageContainer
      title="API Key 管理"
      subTitle="管理 API 访问密钥，控制访问权限和速率限制"
    >
      <ProTable<ApiKeyInfo>
        headerTitle="API Key 列表"
        actionRef={actionRef}
        rowKey="id"
        search={{
          labelWidth: 120,
          defaultCollapsed: false,
        }}
        toolBarRender={() => [
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            创建 API Key
          </Button>,
        ]}
        request={async (params) => {
          const { current, pageSize, ...restParams } = params;
          const response = await getApiKeys({
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
        scroll={{ x: 1600 }}
      />

      <Modal
        title="创建 API Key"
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnClose
        okText={createdKey ? '完成' : '创建'}
        cancelButtonProps={{ style: { display: createdKey ? 'none' : undefined } }}
      >
        {createdKey ? (
          <div>
            <Alert
              message="API Key 创建成功"
              description="请立即复制并保存，此密钥只会显示一次"
              type="success"
              showIcon
              style={{ marginBottom: 16 }}
            />
            <Descriptions bordered column={1}>
              <Descriptions.Item label="API Key">
                <Space>
                  <code style={{ 
                    background: '#f5f5f5', 
                    padding: '8px 12px', 
                    borderRadius: 4,
                    fontSize: 14,
                    wordBreak: 'break-all',
                  }}>
                    {createdKey}
                  </code>
                  <Button
                    type="primary"
                    icon={<CopyOutlined />}
                    onClick={handleCopyKey}
                  >
                    复制
                  </Button>
                </Space>
              </Descriptions.Item>
            </Descriptions>
          </div>
        ) : (
          <Form
            form={form}
            layout="vertical"
            initialValues={{ permissions: ['chat:read'], rateLimit: 60 }}
          >
            <Form.Item
              name="name"
              label="名称"
              rules={[{ required: true, message: '请输入 API Key 名称' }]}
            >
              <Input placeholder="例如：生产环境密钥" />
            </Form.Item>

            <Form.Item
              name="userId"
              label="所属用户"
              rules={[{ required: true, message: '请选择所属用户' }]}
            >
              <Select placeholder="请选择用户">
                <Option value="user1">user1</Option>
                <Option value="user2">user2</Option>
              </Select>
            </Form.Item>

            <Form.Item
              name="permissions"
              label="权限"
              rules={[{ required: true, message: '请选择权限' }]}
            >
              <Select mode="multiple" placeholder="请选择权限">
                <Option value="chat:read">对话读取</Option>
                <Option value="chat:write">对话写入</Option>
                <Option value="models:read">模型读取</Option>
                <Option value="billing:read">账单读取</Option>
              </Select>
            </Form.Item>

            <Form.Item
              name="rateLimit"
              label="速率限制 (请求/分钟)"
            >
              <Input type="number" placeholder="0 表示无限制" />
            </Form.Item>

            <Form.Item
              name="expiresAt"
              label="过期时间"
            >
              <Input type="datetime-local" />
            </Form.Item>

            <Form.Item
              name="description"
              label="描述"
            >
              <TextArea rows={3} placeholder="可选描述信息" />
            </Form.Item>
          </Form>
        )}
      </Modal>
    </PageContainer>
  );
};

export default ApiKeyList;
