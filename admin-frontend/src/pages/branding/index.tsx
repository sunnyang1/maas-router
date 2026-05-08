import React, { useState, useEffect } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import {
  Card,
  Form,
  Input,
  Button,
  Select,
  Space,
  message,
  Typography,
  Divider,
  ColorPicker,
  Switch,
  Tabs,
  Spin,
} from '@ant-design/icons';
import type { Color } from 'antd/es/color-picker';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

interface BrandingFormData {
  site_name: string;
  logo_url: string;
  favicon_url: string;
  primary_color: string;
  footer_text: string;
  custom_css: string;
  about_page: string;
  announcement: string;
  contact_email: string;
  theme: 'light' | 'dark' | 'system';
}

const defaultValues: BrandingFormData = {
  site_name: 'MaaS Router',
  logo_url: '',
  favicon_url: '',
  primary_color: '#1677ff',
  footer_text: '',
  custom_css: '',
  about_page: '',
  announcement: '',
  contact_email: '',
  theme: 'system',
};

const BrandingPage: React.FC = () => {
  const [form] = Form.useForm<BrandingFormData>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [previewColor, setPreviewColor] = useState<string>(defaultValues.primary_color);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      // TODO: replace with actual API call
      // const res = await getBrandingSettings();
      // form.setFieldsValue(res.data);
      form.setFieldsValue(defaultValues);
      setPreviewColor(defaultValues.primary_color);
    } catch (err) {
      message.error('获取品牌设置失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSettings();
  }, [form]);

  const handleSave = async (values: BrandingFormData) => {
    setSaving(true);
    try {
      // TODO: replace with actual API call
      // await updateBrandingSettings(values);
      message.success('品牌设置已保存');
    } catch (err) {
      message.error('保存品牌设置失败');
    } finally {
      setSaving(false);
    }
  };

  const handleColorChange = (color: Color) => {
    const hex = color.toHexString();
    setPreviewColor(hex);
    form.setFieldValue('primary_color', hex);
  };

  const currentValues = Form.useWatch([], form) || defaultValues;

  return (
    <PageContainer
      header={{
        title: '品牌设置',
        subTitle: '自定义站点外观和品牌信息',
      }}
    >
      <Spin spinning={loading}>
        <Tabs
          items={[
            {
              key: 'edit',
              label: '编辑设置',
              children: (
                <Card>
                  <Form<BrandingFormData>
                    form={form}
                    layout="vertical"
                    initialValues={defaultValues}
                    onFinish={handleSave}
                    style={{ maxWidth: 720 }}
                  >
                    <Title level={5}>基本信息</Title>

                    <Form.Item
                      label="站点名称"
                      name="site_name"
                      rules={[{ required: true, message: '请输入站点名称' }]}
                    >
                      <Input placeholder="请输入站点名称" maxLength={50} />
                    </Form.Item>

                    <Form.Item label="Logo URL" name="logo_url">
                      <Input placeholder="https://example.com/logo.png" />
                    </Form.Item>

                    <Form.Item label="Favicon URL" name="favicon_url">
                      <Input placeholder="https://example.com/favicon.ico" />
                    </Form.Item>

                    <Form.Item label="联系邮箱" name="contact_email">
                      <Input placeholder="admin@example.com" type="email" />
                    </Form.Item>

                    <Divider />

                    <Title level={5}>外观设置</Title>

                    <Form.Item label="主题色" name="primary_color">
                      <ColorPicker
                        showText
                        onChange={handleColorChange}
                        format="hex"
                      />
                    </Form.Item>

                    <Form.Item label="主题模式" name="theme">
                      <Select
                        options={[
                          { label: '浅色', value: 'light' },
                          { label: '深色', value: 'dark' },
                          { label: '跟随系统', value: 'system' },
                        ]}
                      />
                    </Form.Item>

                    <Divider />

                    <Title level={5}>内容设置</Title>

                    <Form.Item label="公告" name="announcement">
                      <Input placeholder="输入公告内容，留空则不显示" />
                    </Form.Item>

                    <Form.Item label="页脚文本" name="footer_text">
                      <Input placeholder="页脚显示的文本" />
                    </Form.Item>

                    <Form.Item label="关于页面 (Markdown)" name="about_page">
                      <TextArea
                        rows={8}
                        placeholder="使用 Markdown 语法编写关于页面内容"
                      />
                    </Form.Item>

                    <Form.Item label="自定义 CSS" name="custom_css">
                      <TextArea
                        rows={6}
                        placeholder="输入自定义 CSS 样式"
                        style={{ fontFamily: 'monospace' }}
                      />
                    </Form.Item>

                    <Form.Item>
                      <Space>
                        <Button type="primary" htmlType="submit" loading={saving}>
                          保存设置
                        </Button>
                        <Button onClick={() => form.resetFields()}>
                          重置
                        </Button>
                      </Space>
                    </Form.Item>
                  </Form>
                </Card>
              ),
            },
            {
              key: 'preview',
              label: '预览',
              children: (
                <Card>
                  <Title level={5}>预览效果</Title>
                  <Paragraph type="secondary">
                    以下为当前设置的实际效果预览
                  </Paragraph>

                  <Divider />

                  {/* Announcement Bar Preview */}
                  {currentValues.announcement && (
                    <div
                      style={{
                        backgroundColor: previewColor,
                        color: '#fff',
                        padding: '8px 16px',
                        textAlign: 'center',
                        borderRadius: '4px 4px 0 0',
                        marginBottom: 0,
                      }}
                    >
                      <Text style={{ color: '#fff' }}>
                        {currentValues.announcement}
                      </Text>
                    </div>
                  )}

                  {/* Header Preview */}
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '12px 24px',
                      borderBottom: '1px solid #f0f0f0',
                      backgroundColor: '#fff',
                    }}
                  >
                    <Space>
                      {currentValues.logo_url ? (
                        <img
                          src={currentValues.logo_url}
                          alt="logo"
                          style={{ height: 32 }}
                        />
                      ) : (
                        <div
                          style={{
                            width: 32,
                            height: 32,
                            borderRadius: 6,
                            backgroundColor: previewColor,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: '#fff',
                            fontWeight: 'bold',
                          }}
                        >
                          M
                        </div>
                      )}
                      <Title level={4} style={{ margin: 0 }}>
                        {currentValues.site_name || 'MaaS Router'}
                      </Title>
                    </Space>
                    <Space>
                      <span
                        style={{
                          padding: '4px 12px',
                          borderRadius: 4,
                          backgroundColor: previewColor,
                          color: '#fff',
                          fontSize: 14,
                        }}
                      >
                        控制台
                      </span>
                      <span
                        style={{
                          padding: '4px 12px',
                          borderRadius: 4,
                          fontSize: 14,
                          color: '#666',
                        }}
                      >
                        API密钥
                      </span>
                    </Space>
                  </div>

                  {/* Footer Preview */}
                  <div
                    style={{
                      padding: '12px 24px',
                      borderTop: '1px solid #f0f0f0',
                      backgroundColor: '#fafafa',
                      textAlign: 'center',
                    }}
                  >
                    <Text type="secondary">
                      {currentValues.footer_text || 'Powered by MaaS Router'}
                    </Text>
                    {currentValues.contact_email && (
                      <div>
                        <Text type="secondary">
                          联系邮箱: {currentValues.contact_email}
                        </Text>
                      </div>
                    )}
                  </div>

                  {/* Theme Preview */}
                  <Divider />
                  <Title level={5}>主题色预览</Title>
                  <Space size="large">
                    <div>
                      <Text type="secondary">主色</Text>
                      <div
                        style={{
                          width: 60,
                          height: 60,
                          backgroundColor: previewColor,
                          borderRadius: 8,
                          marginTop: 4,
                        }}
                      />
                    </div>
                    <div>
                      <Text type="secondary">浅色变体</Text>
                      <div
                        style={{
                          width: 60,
                          height: 60,
                          backgroundColor: previewColor + '20',
                          borderRadius: 8,
                          marginTop: 4,
                          border: `1px solid ${previewColor}40`,
                        }}
                      />
                    </div>
                    <div>
                      <Text type="secondary">按钮</Text>
                      <div
                        style={{
                          width: 80,
                          height: 36,
                          backgroundColor: previewColor,
                          borderRadius: 6,
                          marginTop: 4,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          color: '#fff',
                          fontSize: 14,
                        }}
                      >
                        按钮
                      </div>
                    </div>
                  </Space>
                </Card>
              ),
            },
          ]}
        />
      </Spin>
    </PageContainer>
  );
};

export default BrandingPage;
