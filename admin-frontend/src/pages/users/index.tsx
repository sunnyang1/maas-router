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
  Avatar,
  Tooltip,
  Table,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  LockOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import { history } from '@umijs/max';
import type { UserInfo } from '@/services/api';
import { getUsers, createUser, updateUser, deleteUser } from '@/services/api';

const { Option } = Select;

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

const UserList: React.FC = () => {
  const actionRef = useRef<ActionType>();
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('新增用户');
  const [editingUser, setEditingUser] = useState<UserInfo | null>(null);
  const [form] = Form.useForm();

  const columns: ProColumns<UserInfo>[] = [
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      width: 200,
      fixed: 'left',
      render: (_, record) => (
        <Space>
          <Avatar src={record.avatar} style={{ backgroundColor: '#1890ff' }}>
            {record.username.charAt(0).toUpperCase()}
          </Avatar>
          <div>
            <div style={{ fontWeight: 500 }}>{record.username}</div>
            <div style={{ fontSize: 12, color: '#999' }}>{record.email}</div>
          </div>
        </Space>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      valueEnum: {
        admin: { text: '管理员', status: 'Error' },
        user: { text: '普通用户', status: 'Processing' },
        viewer: { text: '访客', status: 'Success' },
      },
      render: (_, record) => (
        <Tag color={UserRoleMap[record.role]?.color}>
          {UserRoleMap[record.role]?.text}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      valueEnum: {
        active: { text: '正常', status: 'Success' },
        inactive: { text: '未激活', status: 'Default' },
        banned: { text: '已禁用', status: 'Error' },
      },
      render: (_, record) => (
        <Tag color={UserStatusMap[record.status]?.color}>
          {UserStatusMap[record.status]?.text}
        </Tag>
      ),
    },
    {
      title: '账户余额',
      dataIndex: 'balance',
      key: 'balance',
      width: 120,
      hideInSearch: true,
      render: (_, record) => (
        <span style={{ color: '#52c41a', fontWeight: 500 }}>
          ¥{record.balance?.toFixed(2) || '0.00'}
        </span>
      ),
    },
    {
      title: '配额使用',
      dataIndex: 'quotaUsed',
      key: 'quotaUsed',
      width: 150,
      hideInSearch: true,
      render: (_, record) => {
        const used = record.quotaUsed || 0;
        const limit = record.quotaLimit || 0;
        const percent = limit > 0 ? Math.round((used / limit) * 100) : 0;
        return (
          <Tooltip title={`${used.toLocaleString()} / ${limit > 0 ? limit.toLocaleString() : '无限制'}`}>
            <div style={{ width: 100 }}>
              <div style={{ fontSize: 12, marginBottom: 4 }}>
                {percent}% ({used.toLocaleString()})
              </div>
              <div
                style={{
                  width: '100%',
                  height: 6,
                  backgroundColor: '#f0f0f0',
                  borderRadius: 3,
                }}
              >
                <div
                  style={{
                    width: `${percent}%`,
                    height: '100%',
                    backgroundColor: percent > 80 ? '#ff4d4f' : '#52c41a',
                    borderRadius: 3,
                    transition: 'width 0.3s',
                  }}
                />
              </div>
            </div>
          </Tooltip>
        );
      },
    },
    {
      title: '手机号',
      dataIndex: 'phone',
      key: 'phone',
      width: 130,
      hideInSearch: true,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      hideInSearch: true,
      sorter: true,
      valueType: 'dateTime',
    },
    {
      title: '最后登录',
      dataIndex: 'lastLoginAt',
      key: 'lastLoginAt',
      width: 170,
      hideInSearch: true,
      valueType: 'dateTime',
      render: (_, record) => record.lastLoginAt || '-',
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      hideInSearch: true,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="text"
            icon={<EyeOutlined />}
            onClick={() => history.push(`/users/${record.id}`)}
          >
            详情
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
    setEditingUser(null);
    setModalTitle('新增用户');
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: UserInfo) => {
    setEditingUser(record);
    setModalTitle('编辑用户');
    form.setFieldsValue({
      username: record.username,
      email: record.email,
      phone: record.phone,
      role: record.role,
      status: record.status,
      quotaLimit: record.quotaLimit,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    try {
      const response = await deleteUser(id);
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

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      if (editingUser) {
        const response = await updateUser(editingUser.id, values);
        if (response.success) {
          message.success('更新成功');
          setModalVisible(false);
          actionRef.current?.reload();
        } else {
          message.error(response.message || '更新失败');
        }
      } else {
        const response = await createUser(values);
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
      title="用户管理"
      subTitle="管理系统用户，包括创建、编辑和删除用户"
    >
      <ProTable<UserInfo>
        headerTitle="用户列表"
        actionRef={actionRef}
        rowKey="id"
        search={{
          labelWidth: 120,
          defaultCollapsed: false,
        }}
        toolBarRender={() => [
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增用户
          </Button>,
        ]}
        request={async (params, sort, filter) => {
          const { current, pageSize, ...restParams } = params;
          const response = await getUsers({
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
        scroll={{ x: 1400 }}
        rowSelection={{
          selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT],
        }}
      />

      <Modal
        title={modalTitle}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ role: 'user', status: 'active' }}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少3个字符' },
              { max: 20, message: '用户名最多20个字符' },
            ]}
          >
            <Input placeholder="请输入用户名" disabled={!!editingUser} />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '请输入有效的邮箱地址' },
            ]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[
                { required: true, message: '请输入密码' },
                { min: 6, message: '密码至少6个字符' },
              ]}
            >
              <Input.Password placeholder="请输入密码" />
            </Form.Item>
          )}

          <Form.Item name="phone" label="手机号">
            <Input placeholder="请输入手机号" />
          </Form.Item>

          <Form.Item
            name="role"
            label="角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Select placeholder="请选择角色">
              <Option value="admin">管理员</Option>
              <Option value="user">普通用户</Option>
              <Option value="viewer">访客</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select placeholder="请选择状态">
              <Option value="active">
                <Tag color="success">正常</Tag>
              </Option>
              <Option value="inactive">
                <Tag color="default">未激活</Tag>
              </Option>
              <Option value="banned">
                <Tag color="error">已禁用</Tag>
              </Option>
            </Select>
          </Form.Item>

          <Form.Item name="quotaLimit" label="配额限制">
            <Input type="number" placeholder="请输入配额限制（0表示无限制）" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default UserList;
