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
  InputNumber,
  Card,
  Row,
  Col,
  Switch,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  NodeIndexOutlined,
} from '@ant-design/icons';
import type { RoutingRule } from '@/services/api';
import { getRoutingRules, createRoutingRule, updateRoutingRule, deleteRoutingRule } from '@/services/api';

const { Option } = Select;
const { TextArea } = Input;

const RoutingList: React.FC = () => {
  const actionRef = useRef<ActionType>();
  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState('新增路由规则');
  const [editingRule, setEditingRule] = useState<RoutingRule | null>(null);
  const [form] = Form.useForm();

  const columns: ProColumns<RoutingRule>[] = [
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: true,
      render: (priority: number) => (
        <Tag color={priority <= 3 ? 'red' : priority <= 6 ? 'orange' : 'default'}>
          {priority}
        </Tag>
      ),
    },
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      fixed: 'left',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 250,
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      valueEnum: {
        active: { text: '启用', status: 'Success' },
        inactive: { text: '禁用', status: 'Default' },
      },
      render: (_, record) => (
        <Tag color={record.status === 'active' ? 'success' : 'default'}>
          {record.status === 'active' ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '条件',
      dataIndex: 'conditions',
      key: 'conditions',
      width: 300,
      hideInSearch: true,
      render: (conditions: any[]) => (
        <Space direction="vertical" size="small" style={{ width: '100%' }}>
          {conditions?.map((cond, index) => (
            <Tag key={index} size="small">
              {cond.field} {cond.operator} {cond.value}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '动作',
      dataIndex: 'action',
      key: 'action',
      width: 200,
      hideInSearch: true,
      render: (action: any) => {
        const actionTypeMap: Record<string, string> = {
          group: '转发到分组',
          account: '转发到账号',
          model: '指定模型',
          fallback: 'Fallback',
        };
        return (
          <Tag color="blue">
            {actionTypeMap[action?.type] || action?.type}: {action?.target}
          </Tag>
        );
      },
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
      width: 200,
      hideInSearch: true,
      render: (_, record) => (
        <Space size="small">
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
    setEditingRule(null);
    setModalTitle('新增路由规则');
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: RoutingRule) => {
    setEditingRule(record);
    setModalTitle('编辑路由规则');
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      priority: record.priority,
      status: record.status,
      conditions: record.conditions,
      actionType: record.action?.type,
      actionTarget: record.action?.target,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    try {
      const response = await deleteRoutingRule(id);
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
      const data = {
        ...values,
        action: {
          type: values.actionType,
          target: values.actionTarget,
        },
      };
      delete data.actionType;
      delete data.actionTarget;

      if (editingRule) {
        const response = await updateRoutingRule(editingRule.id, data);
        if (response.success) {
          message.success('更新成功');
          setModalVisible(false);
          actionRef.current?.reload();
        } else {
          message.error(response.message || '更新失败');
        }
      } else {
        const response = await createRoutingRule(data);
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
      title="路由规则配置"
      subTitle="配置请求路由规则，实现智能负载均衡和故障转移"
    >
      <Card style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]}>
          <Col span={24}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <NodeIndexOutlined style={{ fontSize: 24, color: '#1890ff' }} />
              <div>
                <div style={{ fontWeight: 500 }}>路由规则说明</div>
                <div style={{ color: '#666', fontSize: 12 }}>
                  路由规则按优先级顺序匹配，数字越小优先级越高。请求会依次匹配规则，第一个匹配的规则将被执行。
                </div>
              </div>
            </div>
          </Col>
        </Row>
      </Card>

      <ProTable<RoutingRule>
        headerTitle="路由规则列表"
        actionRef={actionRef}
        rowKey="id"
        search={{
          labelWidth: 120,
          defaultCollapsed: false,
        }}
        toolBarRender={() => [
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增规则
          </Button>,
        ]}
        request={async (params) => {
          const { current, pageSize, ...restParams } = params;
          const response = await getRoutingRules({
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
        scroll={{ x: 1300 }}
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
          initialValues={{ priority: 10, status: 'active' }}
        >
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder="例如：VIP用户优先路由" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea rows={2} placeholder="可选描述信息" />
          </Form.Item>

          <Form.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: '请输入优先级' }]}
          >
            <InputNumber
              min={1}
              max={100}
              style={{ width: '100%' }}
              placeholder="数字越小优先级越高"
            />
          </Form.Item>

          <Card title="匹配条件" size="small" style={{ marginBottom: 16 }}>
            <Form.List name="conditions">
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Row key={key} gutter={[8, 8]} align="middle" style={{ marginBottom: 8 }}>
                      <Col span={7}>
                        <Form.Item
                          {...restField}
                          name={[name, 'field']}
                          rules={[{ required: true, message: '请选择字段' }]}
                          noStyle
                        >
                          <Select placeholder="字段">
                            <Option value="userId">用户ID</Option>
                            <Option value="model">模型</Option>
                            <Option value="apiKey">API Key</Option>
                            <Option value="sourceIp">来源IP</Option>
                            <Option value="groupId">分组ID</Option>
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={6}>
                        <Form.Item
                          {...restField}
                          name={[name, 'operator']}
                          rules={[{ required: true, message: '请选择操作符' }]}
                          noStyle
                        >
                          <Select placeholder="操作符">
                            <Option value="eq">等于</Option>
                            <Option value="ne">不等于</Option>
                            <Option value="contains">包含</Option>
                            <Option value="startsWith">开头是</Option>
                            <Option value="in">在列表中</Option>
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={9}>
                        <Form.Item
                          {...restField}
                          name={[name, 'value']}
                          rules={[{ required: true, message: '请输入值' }]}
                          noStyle
                        >
                          <Input placeholder="值" />
                        </Form.Item>
                      </Col>
                      <Col span={2}>
                        <Button type="text" danger onClick={() => remove(name)}>
                          删除
                        </Button>
                      </Col>
                    </Row>
                  ))}
                  <Button type="dashed" onClick={() => add()} block>
                    添加条件
                  </Button>
                </>
              )}
            </Form.List>
          </Card>

          <Card title="执行动作" size="small" style={{ marginBottom: 16 }}>
            <Row gutter={[16, 16]}>
              <Col span={12}>
                <Form.Item
                  name="actionType"
                  label="动作类型"
                  rules={[{ required: true, message: '请选择动作类型' }]}
                >
                  <Select placeholder="请选择">
                    <Option value="group">转发到分组</Option>
                    <Option value="account">转发到账号</Option>
                    <Option value="model">指定模型</Option>
                    <Option value="fallback">Fallback</Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="actionTarget"
                  label="目标"
                  rules={[{ required: true, message: '请输入目标' }]}
                >
                  <Input placeholder="例如：group-1 或 gpt-4" />
                </Form.Item>
              </Col>
            </Row>
          </Card>

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

export default RoutingList;
