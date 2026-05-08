import React, { useEffect, useState } from 'react';
import { PageContainer, ProCard } from '@ant-design/pro-components';
import {
  Card,
  Form,
  Input,
  Button,
  message,
  Tabs,
  Switch,
  InputNumber,
  Select,
  Descriptions,
  Tag,
  Space,
  Row,
  Col,
  Statistic,
  Progress,
  Timeline,
} from 'antd';
import {
  SettingOutlined,
  SafetyOutlined,
  DatabaseOutlined,
  GlobalOutlined,
  InfoCircleOutlined,
  SaveOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { getSystemConfigs, updateSystemConfig, getSystemStatus } from '@/services/api';
import type { SystemConfig } from '@/services/api';
import dayjs from 'dayjs';

const { TabPane } = Tabs;
const { TextArea } = Input;

const SystemSettings: React.FC = () => {
  const [configs, setConfigs] = useState<SystemConfig[]>([]);
  const [systemStatus, setSystemStatus] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [generalForm] = Form.useForm();
  const [securityForm] = Form.useForm();
  const [rateLimitForm] = Form.useForm();

  useEffect(() => {
    fetchConfigs();
    fetchSystemStatus();
  }, []);

  const fetchConfigs = async () => {
    try {
      const response = await getSystemConfigs();
      if (response.success) {
        setConfigs(response.data);
        // 初始化表单值
        const generalValues: any = {};
        const securityValues: any = {};
        const rateLimitValues: any = {};
        
        response.data.forEach(config => {
          if (config.category === 'general') {
            generalValues[config.key] = config.value;
          } else if (config.category === 'security') {
            securityValues[config.key] = config.value;
          } else if (config.category === 'rate_limit') {
            rateLimitValues[config.key] = config.value;
          }
        });
        
        generalForm.setFieldsValue(generalValues);
        securityForm.setFieldsValue(securityValues);
        rateLimitForm.setFieldsValue(rateLimitValues);
      }
    } catch (error) {
      message.error('获取系统配置失败');
    }
  };

  const fetchSystemStatus = async () => {
    try {
      const response = await getSystemStatus();
      if (response.success) {
        setSystemStatus(response.data);
      }
    } catch (error) {
      console.error('Failed to fetch system status:', error);
    }
  };

  const handleSaveGeneral = async () => {
    try {
      const values = await generalForm.validateFields();
      setLoading(true);
      for (const [key, value] of Object.entries(values)) {
        await updateSystemConfig(key, value);
      }
      message.success('保存成功');
      fetchConfigs();
    } catch (error) {
      message.error('保存失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveSecurity = async () => {
    try {
      const values = await securityForm.validateFields();
      setLoading(true);
      for (const [key, value] of Object.entries(values)) {
        await updateSystemConfig(key, value);
      }
      message.success('保存成功');
      fetchConfigs();
    } catch (error) {
      message.error('保存失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveRateLimit = async () => {
    try {
      const values = await rateLimitForm.validateFields();
      setLoading(true);
      for (const [key, value] of Object.entries(values)) {
        await updateSystemConfig(key, value);
      }
      message.success('保存成功');
      fetchConfigs();
    } catch (error) {
      message.error('保存失败');
    } finally {
      setLoading(false);
    }
  };

  const formatUptime = (uptime: string) => {
    const seconds = parseInt(uptime);
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days}天 ${hours}小时 ${minutes}分钟`;
  };

  return (
    <PageContainer
      title="系统设置"
      subTitle="配置系统参数和查看运行状态"
    >
      <Tabs defaultActiveKey="general">
        <TabPane
          tab={
            <span>
              <SettingOutlined />
              基础设置
            </span>
          }
          key="general"
        >
          <ProCard>
            <Form
              form={generalForm}
              layout="vertical"
              style={{ maxWidth: 600 }}
            >
              <Form.Item
                name="siteName"
                label="站点名称"
                rules={[{ required: true, message: '请输入站点名称' }]}
              >
                <Input placeholder="MaaS-Router Admin" />
              </Form.Item>

              <Form.Item
                name="siteDescription"
                label="站点描述"
              >
                <TextArea rows={2} placeholder="站点描述信息" />
              </Form.Item>

              <Form.Item
                name="contactEmail"
                label="联系邮箱"
                rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
              >
                <Input placeholder="admin@example.com" />
              </Form.Item>

              <Form.Item
                name="defaultLanguage"
                label="默认语言"
              >
                <Select>
                  <Select.Option value="zh-CN">简体中文</Select.Option>
                  <Select.Option value="en-US">English</Select.Option>
                </Select>
              </Form.Item>

              <Form.Item
                name="enableRegistration"
                label="允许注册"
                valuePropName="checked"
              >
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>

              <Form.Item>
                <Button type="primary" icon={<SaveOutlined />} onClick={handleSaveGeneral} loading={loading}>
                  保存设置
                </Button>
              </Form.Item>
            </Form>
          </ProCard>
        </TabPane>

        <TabPane
          tab={
            <span>
              <SafetyOutlined />
              安全设置
            </span>
          }
          key="security"
        >
          <ProCard>
            <Form
              form={securityForm}
              layout="vertical"
              style={{ maxWidth: 600 }}
            >
              <Form.Item
                name="maxLoginAttempts"
                label="最大登录尝试次数"
                rules={[{ required: true }]}
              >
                <InputNumber min={1} max={10} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item
                name="passwordMinLength"
                label="密码最小长度"
                rules={[{ required: true }]}
              >
                <InputNumber min={6} max={32} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item
                name="sessionTimeout"
                label="会话超时时间 (分钟)"
                rules={[{ required: true }]}
              >
                <InputNumber min={5} max={1440} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item
                name="enableTwoFactor"
                label="启用双因素认证"
                valuePropName="checked"
              >
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>

              <Form.Item
                name="enableAuditLog"
                label="启用审计日志"
                valuePropName="checked"
              >
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>

              <Form.Item>
                <Button type="primary" icon={<SaveOutlined />} onClick={handleSaveSecurity} loading={loading}>
                  保存设置
                </Button>
              </Form.Item>
            </Form>
          </ProCard>
        </TabPane>

        <TabPane
          tab={
            <span>
              <GlobalOutlined />
              限流设置
            </span>
          }
          key="rate_limit"
        >
          <ProCard>
            <Form
              form={rateLimitForm}
              layout="vertical"
              style={{ maxWidth: 600 }}
            >
              <Form.Item
                name="globalRateLimit"
                label="全局速率限制 (请求/分钟)"
                rules={[{ required: true }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item
                name="userRateLimit"
                label="用户速率限制 (请求/分钟)"
                rules={[{ required: true }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item
                name="apiKeyRateLimit"
                label="API Key 速率限制 (请求/分钟)"
                rules={[{ required: true }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>

              <Form.Item
                name="enableRateLimit"
                label="启用限流"
                valuePropName="checked"
              >
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>

              <Form.Item>
                <Button type="primary" icon={<SaveOutlined />} onClick={handleSaveRateLimit} loading={loading}>
                  保存设置
                </Button>
              </Form.Item>
            </Form>
          </ProCard>
        </TabPane>

        <TabPane
          tab={
            <span>
              <DatabaseOutlined />
              系统状态
            </span>
          }
          key="status"
        >
          <ProCard>
            {systemStatus && (
              <>
                <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
                  <Col xs={24} sm={12} lg={6}>
                    <Card>
                      <Statistic
                        title="系统版本"
                        value={systemStatus.version}
                      />
                    </Card>
                  </Col>
                  <Col xs={24} sm={12} lg={6}>
                    <Card>
                      <Statistic
                        title="运行时间"
                        value={formatUptime(systemStatus.uptime)}
                      />
                    </Card>
                  </Col>
                  <Col xs={24} sm={12} lg={6}>
                    <Card>
                      <Statistic
                        title="数据库状态"
                        value={systemStatus.database === 'connected' ? '正常' : '异常'}
                        valueStyle={{ color: systemStatus.database === 'connected' ? '#52c41a' : '#ff4d4f' }}
                      />
                    </Card>
                  </Col>
                  <Col xs={24} sm={12} lg={6}>
                    <Card>
                      <Statistic
                        title="缓存状态"
                        value={systemStatus.cache === 'connected' ? '正常' : '异常'}
                        valueStyle={{ color: systemStatus.cache === 'connected' ? '#52c41a' : '#ff4d4f' }}
                      />
                    </Card>
                  </Col>
                </Row>

                <Row gutter={[16, 16]}>
                  <Col xs={24} sm={12}>
                    <Card title="内存使用">
                      <Progress
                        percent={Math.round((systemStatus.memory.used / systemStatus.memory.total) * 100)}
                        status="active"
                        strokeColor={{ from: '#108ee9', to: '#87d068' }}
                      />
                      <div style={{ marginTop: 8, textAlign: 'center' }}>
                        {Math.round(systemStatus.memory.used / 1024 / 1024)} MB / {Math.round(systemStatus.memory.total / 1024 / 1024)} MB
                      </div>
                    </Card>
                  </Col>
                  <Col xs={24} sm={12}>
                    <Card title="CPU 使用率">
                      <Progress
                        percent={systemStatus.cpu.usage}
                        status={systemStatus.cpu.usage > 80 ? 'exception' : 'active'}
                      />
                      <div style={{ marginTop: 8, textAlign: 'center' }}>
                        {systemStatus.cpu.usage}%
                      </div>
                    </Card>
                  </Col>
                </Row>

                <Card title="系统事件" style={{ marginTop: 16 }}>
                  <Timeline
                    items={[
                      {
                        color: 'green',
                        children: `系统启动 - ${dayjs().format('YYYY-MM-DD HH:mm:ss')}`,
                      },
                      {
                        color: 'blue',
                        children: '配置更新 - 限流参数调整',
                      },
                      {
                        color: 'gray',
                        children: '系统维护 - 数据库优化',
                      },
                    ]}
                  />
                </Card>
              </>
            )}

            <div style={{ marginTop: 16, textAlign: 'center' }}>
              <Button icon={<ReloadOutlined />} onClick={fetchSystemStatus}>
                刷新状态
              </Button>
            </div>
          </ProCard>
        </TabPane>
      </Tabs>
    </PageContainer>
  );
};

export default SystemSettings;
