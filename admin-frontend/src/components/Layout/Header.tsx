/**
 * 管理后台 Header 组件
 * 包含 Logo、导航菜单、用户操作、主题切换等功能
 * 支持移动端响应式布局
 */
import React, { useState } from 'react';
import {
  Layout,
  Space,
  Badge,
  Avatar,
  Dropdown,
  Menu,
  Switch,
  Tooltip,
  Typography,
  Input,
  Button,
  Drawer,
} from 'antd';
import {
  BellOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  MoonOutlined,
  SunOutlined,
  SearchOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  GlobalOutlined,
  MenuOutlined,
} from '@ant-design/icons';

const { Header: AntHeader } = Layout;
const { Text } = Typography;
const { Search } = Input;

/**
 * Header 组件属性
 */
interface HeaderProps {
  /** 是否折叠侧边栏 */
  collapsed: boolean;
  /** 切换侧边栏回调 */
  onCollapse: (collapsed: boolean) => void;
  /** 当前主题 */
  theme: 'light' | 'dark';
  /** 切换主题回调 */
  onThemeChange: (theme: 'light' | 'dark') => void;
  /** 当前语言 */
  locale: string;
  /** 切换语言回调 */
  onLocaleChange: (locale: string) => void;
  /** 用户信息 */
  user?: {
    name: string;
    avatar?: string;
    email?: string;
  };
  /** 是否为移动端 */
  isMobile?: boolean;
  /** 移动端菜单点击回调 */
  onMobileMenuClick?: () => void;
}

/**
 * Header 组件
 */
const Header: React.FC<HeaderProps> = ({
  collapsed,
  onCollapse,
  theme,
  onThemeChange,
  locale,
  onLocaleChange,
  user = { name: '管理员' },
  isMobile = false,
  onMobileMenuClick,
}) => {
  // 通知数量
  const [notificationCount] = useState(5);
  // 搜索框可见性
  const [searchVisible, setSearchVisible] = useState(false);
  // 搜索抽屉可见性（移动端）
  const [searchDrawerVisible, setSearchDrawerVisible] = useState(false);

  /**
   * 用户菜单项
   */
  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人中心',
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '系统设置',
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
    },
  ];

  /**
   * 语言菜单项
   */
  const languageMenuItems = [
    {
      key: 'zh-CN',
      label: '简体中文',
      icon: locale === 'zh-CN' ? '✓' : '',
    },
    {
      key: 'en-US',
      label: 'English',
      icon: locale === 'en-US' ? '✓' : '',
    },
  ];

  /**
   * 处理用户菜单点击
   */
  const handleUserMenuClick = ({ key }: { key: string }) => {
    switch (key) {
      case 'profile':
        // 跳转到个人中心
        console.log('跳转到个人中心');
        break;
      case 'settings':
        // 跳转到系统设置
        console.log('跳转到系统设置');
        break;
      case 'logout':
        // 退出登录
        console.log('退出登录');
        break;
    }
  };

  /**
   * 处理语言切换
   */
  const handleLanguageChange = ({ key }: { key: string }) => {
    onLocaleChange(key);
  };

  /**
   * 处理主题切换
   */
  const handleThemeChange = (checked: boolean) => {
    onThemeChange(checked ? 'dark' : 'light');
  };

  /**
   * 处理折叠切换
   */
  const handleCollapse = () => {
    if (isMobile && onMobileMenuClick) {
      onMobileMenuClick();
    } else {
      onCollapse(!collapsed);
    }
  };

  return (
    <AntHeader
      style={{
        padding: isMobile ? '0 16px' : '0 24px',
        background: theme === 'dark' ? '#001529' : '#fff',
        boxShadow: '0 1px 4px rgba(0,21,41,.08)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'sticky',
        top: 0,
        zIndex: 100,
        width: '100%',
      }}
    >
      {/* 左侧区域 */}
      <Space size={isMobile ? 12 : 24}>
        {/* 折叠/菜单按钮 */}
        <div
          onClick={handleCollapse}
          style={{
            cursor: 'pointer',
            fontSize: 18,
            color: theme === 'dark' ? '#fff' : '#000',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          {isMobile ? <MenuOutlined /> : (collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />)}
        </div>

        {/* Logo */}
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 32,
              height: 32,
              background: 'linear-gradient(135deg, #1890ff 0%, #36cfc9 100%)',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
            }}
          >
            <span style={{ color: '#fff', fontWeight: 'bold', fontSize: 16 }}>M</span>
          </div>
          {!isMobile && (
            <Text
              strong
              style={{
                fontSize: 18,
                color: theme === 'dark' ? '#fff' : '#000',
              }}
            >
              MaaS-Router
            </Text>
          )}
        </div>

        {/* 搜索框 - 桌面端 */}
        {!isMobile && searchVisible ? (
          <Search
            placeholder="搜索菜单、功能..."
            allowClear
            autoFocus
            onBlur={() => setSearchVisible(false)}
            onSearch={(value) => console.log('搜索:', value)}
            style={{ width: 250 }}
          />
        ) : !isMobile ? (
          <Tooltip title="搜索">
            <SearchOutlined
              onClick={() => setSearchVisible(true)}
              style={{
                fontSize: 16,
                cursor: 'pointer',
                color: theme === 'dark' ? '#fff' : '#000',
              }}
            />
          </Tooltip>
        ) : null}
      </Space>

      {/* 右侧区域 */}
      <Space size={isMobile ? 8 : 16}>
        {/* 搜索按钮 - 移动端 */}
        {isMobile && (
          <Tooltip title="搜索">
            <SearchOutlined
              onClick={() => setSearchDrawerVisible(true)}
              style={{
                fontSize: 18,
                cursor: 'pointer',
                color: theme === 'dark' ? '#fff' : '#000',
              }}
            />
          </Tooltip>
        )}

        {/* 主题切换 */}
        {!isMobile && (
          <Tooltip title={theme === 'dark' ? '切换到亮色模式' : '切换到暗色模式'}>
            <Switch
              checked={theme === 'dark'}
              onChange={handleThemeChange}
              checkedChildren={<MoonOutlined />}
              unCheckedChildren={<SunOutlined />}
            />
          </Tooltip>
        )}

        {/* 语言切换 */}
        {!isMobile && (
          <Dropdown
            menu={{ items: languageMenuItems, onClick: handleLanguageChange }}
            placement="bottomRight"
          >
            <Space style={{ cursor: 'pointer' }}>
              <GlobalOutlined
                style={{
                  fontSize: 16,
                  color: theme === 'dark' ? '#fff' : '#000',
                }}
              />
              <Text
                style={{
                  color: theme === 'dark' ? '#fff' : '#000',
                }}
              >
                {locale === 'zh-CN' ? '中文' : 'EN'}
              </Text>
            </Space>
          </Dropdown>
        )}

        {/* 通知 */}
        <Tooltip title="通知">
          <Badge count={notificationCount} size="small">
            <BellOutlined
              style={{
                fontSize: 18,
                cursor: 'pointer',
                color: theme === 'dark' ? '#fff' : '#000',
              }}
            />
          </Badge>
        </Tooltip>

        {/* 用户菜单 */}
        <Dropdown
          menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
          placement="bottomRight"
        >
          <Space style={{ cursor: 'pointer', marginLeft: isMobile ? 0 : 8 }}>
            <Avatar
              src={user.avatar}
              icon={!user.avatar && <UserOutlined />}
              style={{ backgroundColor: '#1890ff' }}
              size={isMobile ? 'small' : 'default'}
            />
            {!isMobile && (
              <Text
                style={{
                  color: theme === 'dark' ? '#fff' : '#000',
                  display: 'inline-block',
                  maxWidth: 100,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {user.name}
              </Text>
            )}
          </Space>
        </Dropdown>
      </Space>

      {/* 移动端搜索抽屉 */}
      <Drawer
        placement="top"
        closable={false}
        onClose={() => setSearchDrawerVisible(false)}
        open={searchDrawerVisible}
        height="auto"
        bodyStyle={{ padding: '16px' }}
      >
        <Search
          placeholder="搜索菜单、功能..."
          allowClear
          autoFocus
          enterButton
          onSearch={(value) => {
            console.log('搜索:', value);
            setSearchDrawerVisible(false);
          }}
          onBlur={() => setSearchDrawerVisible(false)}
        />
      </Drawer>
    </AntHeader>
  );
};

export default Header;
