/**
 * 管理后台 Footer 组件
 * 包含版权信息、链接、系统状态等
 */
import React from 'react';
import { Layout, Space, Typography, Tooltip, Badge } from 'antd';
import {
  GithubOutlined,
  QuestionCircleOutlined,
  MailOutlined,
  GlobalOutlined,
  HeartFilled,
  CloudOutlined,
  CheckCircleFilled,
} from '@ant-design/icons';

const { Footer: AntFooter } = Layout;
const { Text, Link } = Typography;

/**
 * Footer 组件属性
 */
interface FooterProps {
  /** 当前主题 */
  theme: 'light' | 'dark';
  /** 系统状态 */
  systemStatus?: 'online' | 'degraded' | 'offline';
  /** 版本号 */
  version?: string;
  /** 构建时间 */
  buildTime?: string;
}

/**
 * Footer 组件
 */
const Footer: React.FC<FooterProps> = ({
  theme,
  systemStatus = 'online',
  version = '1.0.0',
  buildTime = '2024-01-01',
}) => {
  // 获取当前年份
  const currentYear = new Date().getFullYear();

  /**
   * 系统状态配置
   */
  const statusConfig = {
    online: {
      color: '#52c41a',
      text: '系统正常',
      icon: <CheckCircleFilled />,
    },
    degraded: {
      color: '#faad14',
      text: '服务降级',
      icon: <CloudOutlined />,
    },
    offline: {
      color: '#f5222d',
      text: '系统离线',
      icon: <CloudOutlined />,
    },
  };

  const status = statusConfig[systemStatus];

  return (
    <AntFooter
      style={{
        padding: '16px 24px',
        background: theme === 'dark' ? '#001529' : '#f0f2f5',
        borderTop: `1px solid ${theme === 'dark' ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.06)'}`,
        textAlign: 'center',
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 16,
        }}
      >
        {/* 左侧：版权信息 */}
        <div>
          <Text
            type="secondary"
            style={{
              color: theme === 'dark' ? 'rgba(255,255,255,0.45)' : undefined,
            }}
          >
            © {currentYear} MaaS-Router. Made with{' '}
            <HeartFilled style={{ color: '#ff4d4f' }} /> All Rights Reserved.
          </Text>
        </div>

        {/* 中间：链接 */}
        <Space size={24}>
          <Tooltip title="GitHub">
            <Link
              href="https://github.com/your-org/maas-router"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                color: theme === 'dark' ? 'rgba(255,255,255,0.65)' : undefined,
              }}
            >
              <GithubOutlined style={{ fontSize: 16 }} />
            </Link>
          </Tooltip>

          <Tooltip title="帮助文档">
            <Link
              href="/docs"
              style={{
                color: theme === 'dark' ? 'rgba(255,255,255,0.65)' : undefined,
              }}
            >
              <QuestionCircleOutlined style={{ fontSize: 16 }} />
            </Link>
          </Tooltip>

          <Tooltip title="联系我们">
            <Link
              href="mailto:support@maas-router.com"
              style={{
                color: theme === 'dark' ? 'rgba(255,255,255,0.65)' : undefined,
              }}
            >
              <MailOutlined style={{ fontSize: 16 }} />
            </Link>
          </Tooltip>

          <Tooltip title="官方网站">
            <Link
              href="https://maas-router.com"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                color: theme === 'dark' ? 'rgba(255,255,255,0.65)' : undefined,
              }}
            >
              <GlobalOutlined style={{ fontSize: 16 }} />
            </Link>
          </Tooltip>
        </Space>

        {/* 右侧：系统状态和版本 */}
        <Space size={16}>
          {/* 系统状态 */}
          <Tooltip title={`系统状态: ${status.text}`}>
            <Badge
              status={systemStatus === 'online' ? 'success' : systemStatus === 'degraded' ? 'warning' : 'error'}
              text={
                <Text
                  style={{
                    color: theme === 'dark' ? 'rgba(255,255,255,0.65)' : undefined,
                    fontSize: 12,
                  }}
                >
                  {status.text}
                </Text>
              }
            />
          </Tooltip>

          {/* 版本信息 */}
          <Tooltip title={`构建时间: ${buildTime}`}>
            <Text
              type="secondary"
              style={{
                color: theme === 'dark' ? 'rgba(255,255,255,0.45)' : undefined,
                fontSize: 12,
              }}
            >
              v{version}
            </Text>
          </Tooltip>
        </Space>
      </div>
    </AntFooter>
  );
};

export default Footer;
