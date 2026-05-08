/**
 * 卡密管理页面
 * 支持批量生成、查看、导出卡密
 * 响应式表格设计，适配移动端
 */
import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Space,
  Tag,
  Modal,
  Form,
  InputNumber,
  DatePicker,
  message,
  Tooltip,
  Row,
  Col,
  Statistic,
  Grid,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ExportOutlined,
  CopyOutlined,
  StopOutlined,
  KeyOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';

const { useBreakpoint } = Grid;

// 卡密数据类型
interface RedeemCode {
  id: number;
  code: string;
  amount: number;
  status: 'unused' | 'used' | 'expired' | 'disabled';
  batchNo: string;
  remark?: string;
  expiresAt?: string;
  usedAt?: string;
  usedBy?: number;
  createdAt: string;
}

/**
 * 卡密管理页面
 */
const RedeemCodesPage: React.FC = () => {
  const screens = useBreakpoint();
  const isMobile = !screens.md;
  
  // 状态
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<RedeemCode[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [generateModalVisible, setGenerateModalVisible] = useState(false);
  const [generateForm] = Form.useForm();

  // 模拟数据
  const mockData: RedeemCode[] = [
    {
      id: 1,
      code: 'MR2024A1B2C3D4E5',
      amount: 100,
      status: 'unused',
      batchNo: 'B202401150001',
      remark: '春节活动卡密',
      expiresAt: '2024-12-31T23:59:59Z',
      createdAt: '2024-01-15T10:00:00Z',
    },
    {
      id: 2,
      code: 'MR2024F6G7H8I9J0',
      amount: 50,
      status: 'used',
      batchNo: 'B202401150001',
      remark: '春节活动卡密',
      usedAt: '2024-01-16T14:30:00Z',
      usedBy: 10001,
      createdAt: '2024-01-15T10:00:00Z',
    },
    {
      id: 3,
      code: 'MR2024K1L2M3N4O5',
      amount: 200,
      status: 'expired',
      batchNo: 'B202312010001',
      remark: '双十二活动',
      expiresAt: '2023-12-31T23:59:59Z',
      createdAt: '2023-12-01T09:00:00Z',
    },
    {
      id: 4,
      code: 'MR2024P6Q7R8S9T0',
      amount: 100,
      status: 'disabled',
      batchNo: 'B202401150002',
      remark: '测试卡密',
      createdAt: '2024-01-15T11:00:00Z',
    },
  ];

  // 加载数据
  const loadData = async () => {
    setLoading(true);
    try {
      // 模拟 API 调用
      await new Promise(resolve => setTimeout(resolve, 500));
      
      let filtered = [...mockData];
      
      // 搜索过滤
      if (searchKeyword) {
        filtered = filtered.filter(item => 
          item.code.toLowerCase().includes(searchKeyword.toLowerCase()) ||
          item.batchNo.toLowerCase().includes(searchKeyword.toLowerCase())
        );
      }
      
      // 状态过滤
      if (statusFilter) {
        filtered = filtered.filter(item => item.status === statusFilter);
      }
      
      setData(filtered);
      setTotal(filtered.length);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [page, pageSize, searchKeyword, statusFilter]);

  // 复制卡密
  const handleCopyCode = (code: string) => {
    navigator.clipboard.writeText(code);
    message.success('卡密已复制到剪贴板');
  };

  // 禁用卡密
  const handleDisable = (record: RedeemCode) => {
    Modal.confirm({
      title: '确认禁用',
      content: `确定要禁用卡密 ${record.code} 吗？禁用后该卡密将无法使用。`,
      onOk: () => {
        message.success('卡密已禁用');
        loadData();
      },
    });
  };

  // 生成卡密
  const handleGenerate = async (values: any) => {
    try {
      message.success(`成功生成 ${values.count} 个卡密`);
      setGenerateModalVisible(false);
      generateForm.resetFields();
      loadData();
    } catch (error) {
      message.error('生成卡密失败');
    }
  };

  // 导出卡密
  const handleExport = () => {
    message.success('卡密导出成功');
  };

  // 获取状态标签
  const getStatusTag = (status: string) => {
    const statusMap: Record<string, { color: string; icon: React.ReactNode; text: string }> = {
      unused: { color: 'success', icon: <CheckCircleOutlined />, text: '未使用' },
      used: { color: 'default', icon: <CloseCircleOutlined />, text: '已使用' },
      expired: { color: 'warning', icon: <ClockCircleOutlined />, text: '已过期' },
      disabled: { color: 'error', icon: <StopOutlined />, text: '已禁用' },
    };
    const config = statusMap[status] || statusMap.unused;
    return (
      <Tag icon={config.icon} color={config.color}>
        {config.text}
      </Tag>
    );
  };

  // 表格列定义
  const columns: ColumnsType<RedeemCode> = [
    {
      title: '卡密',
      dataIndex: 'code',
      key: 'code',
      render: (code: string) => (
        <Space>
          <span style={{ fontFamily: 'monospace' }}>{code}</span>
          <Tooltip title="复制">
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined />}
              onClick={() => handleCopyCode(code)}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: '面额',
      dataIndex: 'amount',
      key: 'amount',
      width: 100,
      render: (amount: number) => `¥${amount.toFixed(2)}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => getStatusTag(status),
    },
    {
      title: '批次号',
      dataIndex: 'batchNo',
      key: 'batchNo',
      ellipsis: true,
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      ellipsis: true,
    },
    {
      title: '过期时间',
      dataIndex: 'expiresAt',
      key: 'expiresAt',
      render: (date?: string) => date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '永不过期',
    },
    {
      title: '使用时间',
      dataIndex: 'usedAt',
      key: 'usedAt',
      render: (date?: string) => date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          {record.status === 'unused' && (
            <Tooltip title="禁用">
              <Button
                type="text"
                danger
                size="small"
                icon={<StopOutlined />}
                onClick={() => handleDisable(record)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  // 移动端简化列
  const mobileColumns: ColumnsType<RedeemCode> = [
    {
      title: '卡密信息',
      key: 'info',
      render: (_, record) => (
        <div>
          <div style={{ marginBottom: 8 }}>
            <Space>
              <span style={{ fontFamily: 'monospace', fontWeight: 500 }}>{record.code}</span>
              {getStatusTag(record.status)}
            </Space>
          </div>
          <div style={{ color: '#666', fontSize: 12 }}>
            <div>面额: ¥{record.amount.toFixed(2)} | 批次: {record.batchNo}</div>
            {record.remark && <div>备注: {record.remark}</div>}
          </div>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_, record) => (
        <Space direction="vertical" size="small">
          <Button
            type="text"
            size="small"
            icon={<CopyOutlined />}
            onClick={() => handleCopyCode(record.code)}
          >
            复制
          </Button>
          {record.status === 'unused' && (
            <Button
              type="text"
              danger
              size="small"
              icon={<StopOutlined />}
              onClick={() => handleDisable(record)}
            >
              禁用
            </Button>
          )}
        </Space>
      ),
    },
  ];

  // 统计卡片数据
  const stats = [
    { title: '总卡密数', value: 1250, icon: <KeyOutlined /> },
    { title: '未使用', value: 856, color: '#52c41a' },
    { title: '已使用', value: 342, color: '#999' },
    { title: '已过期', value: 42, color: '#faad14' },
    { title: '已禁用', value: 10, color: '#f5222d' },
  ];

  return (
    <div>
      {/* 页面标题 */}
      <div style={{ marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontSize: isMobile ? 20 : 24 }}>卡密管理</h1>
        <p style={{ margin: '8px 0 0', color: '#666' }}>
          管理充值卡密，支持批量生成、导出和禁用操作
        </p>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {stats.map((stat, index) => (
          <Col xs={12} sm={12} md={8} lg={4} key={index}>
            <Card size="small">
              <Statistic
                title={stat.title}
                value={stat.value}
                valueStyle={{ color: stat.color || '#1890ff', fontSize: isMobile ? 20 : 24 }}
                prefix={stat.icon}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* 操作栏 */}
      <Card style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={24} md={12} lg={8}>
            <Input.Search
              placeholder="搜索卡密或批次号"
              allowClear
              enterButton={<><SearchOutlined /> 搜索</>}
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              onSearch={() => loadData()}
            />
          </Col>
          <Col xs={24} sm={24} md={12} lg={16}>
            <Space wrap style={{ justifyContent: isMobile ? 'flex-start' : 'flex-end', width: '100%' }}>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setGenerateModalVisible(true)}
              >
                {isMobile ? '生成' : '生成卡密'}
              </Button>
              <Button
                icon={<ExportOutlined />}
                onClick={handleExport}
              >
                {isMobile ? '导出' : '导出卡密'}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 卡密列表 */}
      <Card>
        <Table
          columns={isMobile ? mobileColumns : columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: !isMobile,
            showQuickJumper: !isMobile,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPage(page);
              setPageSize(pageSize || 10);
            },
            size: isMobile ? 'small' : 'default',
          }}
          scroll={{ x: isMobile ? 300 : 1200 }}
          size={isMobile ? 'small' : 'middle'}
        />
      </Card>

      {/* 生成卡密弹窗 */}
      <Modal
        title="批量生成卡密"
        open={generateModalVisible}
        onCancel={() => setGenerateModalVisible(false)}
        onOk={() => generateForm.submit()}
        okText="生成"
        cancelText="取消"
        width={isMobile ? '90%' : 520}
      >
        <Form
          form={generateForm}
          layout="vertical"
          onFinish={handleGenerate}
          initialValues={{ count: 10, amount: 100 }}
        >
          <Form.Item
            name="amount"
            label="面额（元）"
            rules={[{ required: true, message: '请输入面额' }]}
          >
            <InputNumber
              min={1}
              max={10000}
              style={{ width: '100%' }}
              placeholder="请输入面额"
              prefix="¥"
            />
          </Form.Item>
          
          <Form.Item
            name="count"
            label="生成数量"
            rules={[{ required: true, message: '请输入生成数量' }]}
          >
            <InputNumber
              min={1}
              max={1000}
              style={{ width: '100%' }}
              placeholder="请输入生成数量"
            />
          </Form.Item>
          
          <Form.Item
            name="expiresAt"
            label="过期时间"
          >
            <DatePicker
              showTime
              style={{ width: '100%' }}
              placeholder="选择过期时间（可选）"
            />
          </Form.Item>
          
          <Form.Item
            name="remark"
            label="备注"
          >
            <Input.TextArea
              rows={3}
              placeholder="输入备注信息（可选）"
              maxLength={200}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default RedeemCodesPage;
