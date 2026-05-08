/**
 * 管理后台 Sidebar 组件
 * 包含导航菜单、菜单折叠、权限控制等功能
 * 支持移动端响应式布局
 */
import React, { useState, useEffect } from 'react';
import { Layout, Menu, Badge, Drawer } from 'antd';
import {
  DashboardOutlined,
  ApiOutlined,
  UserOutlined,
  SettingOutlined,
  CreditCardOutlined,
  BarChartOutlined,
  FileTextOutlined,
  SafetyOutlined,
  MessageOutlined,
  CloudServerOutlined,
  GiftOutlined,
  KeyOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';

const { Sider } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

/**
 * Sidebar 组件属性
 */
interface SidebarProps {
  /** 是否折叠 */
  collapsed: boolean;
  /** 当前主题 */
  theme: 'light' | 'dark';
  /** 当前选中的菜单项 */
  selectedKeys?: string[];
  /** 当前展开的子菜单 */
  openKeys?: string[];
  /** 菜单选择回调 */
  onSelect?: (keys: string[]) => void;
  /** 子菜单展开回调 */
  onOpenChange?: (keys: string[]) => void;
  /** 用户权限 */
  permissions?: string[];
  /** 是否为移动端 */
  isMobile?: boolean;
  /** 移动端抽屉是否可见 */
  mobileVisible?: boolean;
  /** 移动端抽屉关闭回调 */
  onMobileClose?: () => void;
  /** 切换折叠状态回调 */
  onCollapse?: (collapsed: boolean) => void;
}

/**
 * Sidebar 组件
 */
const Sidebar: React.FC<SidebarProps> = ({
  collapsed,
  theme,
  selectedKeys = [],
  openKeys = [],
  onSelect,
  onOpenChange,
  permissions = [],
  isMobile = false,
  mobileVisible = false,
  onMobileClose,
  onCollapse,
}) => {
  // 当前展开的子菜单
  const [currentOpenKeys, setCurrentOpenKeys] = useState<string[]>(openKeys);

  // 同步外部 openKeys
  useEffect(() => {
    setCurrentOpenKeys(openKeys);
  }, [openKeys]);

  /**
   * 检查是否有权限
   */
  const hasPermission = (requiredPermissions?: string[]) => {
    if (!requiredPermissions || requiredPermissions.length === 0) {
      return true;
    }
    return requiredPermissions.some(p => permissions.includes(p));
  };

  /**
   * 菜单项定义
   */
  const menuItems: MenuItem[] = [
    {
      key: 'dashboard',
      icon: <DashboardOutlined />,
      label: '仪表盘',
    },
    {
      key: 'api',
      icon: <ApiOutlined />,
      label: 'API 管理',
      children: [
        {
          key: 'api-keys',
          label: 'API 密钥',
        },
        {
          key: 'api-logs',
          label: '调用日志',
        },
        {
          key: 'api-limits',
          label: '限流设置',
        },
      ],
    },
    {
      key: 'router',
      icon: <CloudServerOutlined />,
      label: '模型路由',
      children: [
        {
          key: 'router-config',
          label: '路由配置',
        },
        {
          key: 'router-stats',
          label: '路由统计',
        },
        {
          key: 'model-pool',
          label: '模型池',
        },
      ],
    },
    {
      key: 'users',
      icon: <UserOutlined />,
      label: '用户管理',
      children: [
        {
          key: 'user-list',
          label: '用户列表',
        },
        {
          key: 'user-roles',
          label: '角色权限',
        },
      ],
    },
    {
      key: 'billing',
      icon: <CreditCardOutlined />,
      label: '计费中心',
      children: [
        {
          key: 'billing-overview',
          label: '计费概览',
        },
        {
          key: 'billing-orders',
          label: '订单管理',
          icon: <Badge count={5} size="small" offset={[10, 0]}><span /></Badge>,
        },
        {
          key: 'billing-payments',
          label: '支付记录',
        },
        {
          key: 'billing-invoices',
          label: '发票管理',
        },
      ],
    },
    {
      key: 'redeem-codes',
      icon: <KeyOutlined />,
      label: '卡密管理',
      children: [
        {
          key: 'redeem-codes-list',
          label: '卡密列表',
        },
        {
          key: 'redeem-codes-generate',
          label: '生成卡密',
        },
      ],
    },
    {
      key: 'affiliate',
      icon: <GiftOutlined />,
      label: '返利管理',
      children: [
        {
          key: 'affiliate-overview',
          label: '返利概览',
        },
        {
          key: 'affiliate-records',
          label: '返利记录',
        },
        {
          key: 'affiliate-settings',
          label: '返利设置',
        },
      ],
    },
    {
      key: 'analytics',
      icon: <BarChartOutlined />,
      label: '数据分析',
      children: [
        {
          key: 'analytics-usage',
          label: '使用统计',
        },
        {
          key: 'analytics-cost',
          label: '成本分析',
        },
        {
          key: 'analytics-performance',
          label: '性能监控',
        },
      ],
    },
    {
      key: 'content',
      icon: <FileTextOutlined />,
      label: '内容管理',
      children: [
        {
          key: 'content-docs',
          label: '文档管理',
        },
        {
          key: 'content-faq',
          label: '常见问题',
        },
        {
          key: 'content-announcements',
          label: '公告管理',
        },
      ],
    },
    {
      key: 'security',
      icon: <SafetyOutlined />,
      label: '安全管理',
      children: [
        {
          key: 'security-audit',
          label: '审计日志',
        },
        {
          key: 'security-ips',
          label: 'IP 黑白名单',
        },
        {
          key: 'security-waf',
          label: 'WAF 配置',
        },
      ],
    },
    {
      key: 'messages',
      icon: <MessageOutlined />,
      label: '消息中心',
      children: [
        {
          key: 'messages-list',
          label: (
            <span>
              消息列表
              <Badge count={3} size="small" style={{ marginLeft: 8 }} />
            </span>
          ),
        },
        {
          key: 'messages-templates',
          label: '消息模板',
        },
      ],
    },
    {
      key: 'system',
      icon: <SettingOutlined />,
      label: '系统设置',
      children: [
        {
          key: 'system-general',
          label: '基础设置',
        },
        {
          key: 'system-payment',
          label: '支付配置',
        },
        {
          key: 'system-email',
          label: '邮件配置',
        },
        {
          key: 'system-backup',
          label: '备份恢复',
        },
      ],
    },
  ];

  /**
   * 处理菜单选择
   */
  const handleSelect: MenuProps['onSelect'] = ({ selectedKeys }) => {
    onSelect?.(selectedKeys as string[]);
    // 移动端选择后关闭抽屉
    if (isMobile && onMobileClose) {
      onMobileClose();
    }
  };

  /**
   * 处理子菜单展开
   */
  const handleOpenChange: MenuProps['onOpenChange'] = (keys) => {
    setCurrentOpenKeys(keys as string[]);
    onOpenChange?.(keys as string[]);
  };

  /**
   * 渲染菜单内容
   */
  const renderMenu = () => (
    <Menu
      mode="inline"
      theme={theme}
      selectedKeys={selectedKeys}
      openKeys={collapsed ? [] : currentOpenKeys}
      onSelect={handleSelect}
      onOpenChange={handleOpenChange}
      items={menuItems}
      style={{
        height: '100%',
        borderRight: 0,
        paddingTop: 16,
      }}
    />
  );

  // 移动端使用 Drawer
  if (isMobile) {
    return (
      <Drawer
        placement="left"
        closable={false}
        onClose={onMobileClose}
        open={mobileVisible}
        width={240}
        bodyStyle={{ padding: 0 }}
        headerStyle={{ display: 'none' }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0 24px',
            borderBottom: '1px solid rgba(0,0,0,0.06)',
          }}
        >
          <span style={{ fontSize: 18, fontWeight: 600 }}>MaaS Router</span>
          {onCollapse && (
            <MenuFoldOutlined
              style={{ fontSize: 18, cursor: 'pointer' }}
              onClick={onMobileClose}
            />
          )}
        </div>
        {renderMenu()}
      </Drawer>
    );
  }

  // 桌面端使用 Sider
  return (
    <Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      theme={theme}
      width={240}
      collapsedWidth={80}
      breakpoint="lg"
      onBreakpoint={(broken) => {
        if (broken && onCollapse) {
          onCollapse(true);
        }
      }}
      style={{
        overflow: 'auto',
        height: '100vh',
        position: 'fixed',
        left: 0,
        top: 64,
        bottom: 0,
        boxShadow: '2px 0 8px rgba(0,0,0,0.05)',
      }}
    >
      {renderMenu()}

      {/* 底部版本信息 */}
      {!collapsed && (
        <div
          style={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            padding: '12px 24px',
            textAlign: 'center',
            fontSize: 12,
            color: theme === 'dark' ? 'rgba(255,255,255,0.45)' : 'rgba(0,0,0,0.45)',
            borderTop: `1px solid ${theme === 'dark' ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.06)'}`,
          }}
        >
          <div>MaaS-Router v1.0.0</div>
          <div style={{ fontSize: 11, marginTop: 4 }}>© 2024 All Rights Reserved</div>
        </div>
      )}
    </Sider>
  );
};

export default Sidebar;
