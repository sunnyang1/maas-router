/**
 * MaaS-Router 管理后台入口配置
 * 包含主题切换、国际化支持、全局布局配置
 * 支持移动端响应式布局
 */
import React, { useState, useEffect } from 'react';
import { RuntimeConfig } from '@umijs/max';
import { message, notification, ConfigProvider, theme as antdTheme, Grid } from 'antd';
import { RequestConfig } from '@umijs/max';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import 'dayjs/locale/en';

// 导入自定义布局组件
import { Header, Sidebar, Footer } from './components/Layout';

// 主题配置类型
type ThemeType = 'light' | 'dark';
type LocaleType = 'zh-CN' | 'en-US';

const { useBreakpoint } = Grid;

/**
 * 全局状态类型
 */
interface GlobalState {
  name: string;
  avatar?: string;
  access?: string;
  theme: ThemeType;
  locale: LocaleType;
}

/**
 * 获取存储的主题设置
 */
const getStoredTheme = (): ThemeType => {
  if (typeof window === 'undefined') return 'light';
  return (localStorage.getItem('theme') as ThemeType) || 'light';
};

/**
 * 获取存储的语言设置
 */
const getStoredLocale = (): LocaleType => {
  if (typeof window === 'undefined') return 'zh-CN';
  return (localStorage.getItem('locale') as LocaleType) || 'zh-CN';
};

/**
 * 全局初始化数据配置
 */
export async function getInitialState(): Promise<GlobalState> {
  const token = localStorage.getItem('token');
  const theme = getStoredTheme();
  const locale = getStoredLocale();

  // 设置 dayjs 语言
  dayjs.locale(locale === 'zh-CN' ? 'zh-cn' : 'en');

  if (token) {
    try {
      return {
        name: '管理员',
        avatar: 'https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png',
        access: 'admin',
        theme,
        locale,
      };
    } catch (error) {
      localStorage.removeItem('token');
    }
  }

  return {
    name: '访客',
    access: 'guest',
    theme,
    locale,
  };
}

/**
 * 主题上下文
 */
export const ThemeContext = React.createContext<{
  theme: ThemeType;
  setTheme: (theme: ThemeType) => void;
}>({
  theme: 'light',
  setTheme: () => {},
});

/**
 * 国际化上下文
 */
export const LocaleContext = React.createContext<{
  locale: LocaleType;
  setLocale: (locale: LocaleType) => void;
}>({
  locale: 'zh-CN',
  setLocale: () => {},
});

/**
 * 获取 Ant Design 主题配置
 */
const getAntdTheme = (theme: ThemeType) => {
  const isDark = theme === 'dark';
  return {
    algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: '#1890ff',
      colorSuccess: '#52c41a',
      colorWarning: '#faad14',
      colorError: '#f5222d',
      colorInfo: '#1890ff',
      borderRadius: 6,
      wireframe: false,
    },
    components: {
      Layout: {
        headerBg: isDark ? '#001529' : '#fff',
        siderBg: isDark ? '#001529' : '#fff',
        triggerBg: isDark ? '#002140' : '#fff',
        triggerColor: isDark ? '#fff' : '#000',
      },
      Menu: {
        darkItemBg: '#001529',
        darkSubMenuItemBg: '#000c17',
        darkItemSelectedBg: '#1890ff',
      },
      Table: {
        cellPaddingBlock: 12,
        cellPaddingInline: 12,
      },
      Form: {
        itemMarginBottom: 16,
      },
    },
  };
};

/**
 * 获取 Ant Design 语言包
 */
const getAntdLocale = (locale: LocaleType) => {
  return locale === 'zh-CN' ? zhCN : enUS;
};

/**
 * 根组件包装器
 * 提供主题和国际化上下文
 */
const RootWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [theme, setThemeState] = useState<ThemeType>(getStoredTheme());
  const [locale, setLocaleState] = useState<LocaleType>(getStoredLocale());
  const [collapsed, setCollapsed] = useState(false);

  // 设置主题
  const setTheme = (newTheme: ThemeType) => {
    setThemeState(newTheme);
    localStorage.setItem('theme', newTheme);
    // 更新 body 类名用于全局样式
    document.body.classList.toggle('dark', newTheme === 'dark');
  };

  // 设置语言
  const setLocale = (newLocale: LocaleType) => {
    setLocaleState(newLocale);
    localStorage.setItem('locale', newLocale);
    dayjs.locale(newLocale === 'zh-CN' ? 'zh-cn' : 'en');
    // 刷新页面以应用新语言
    window.location.reload();
  };

  // 初始化主题
  useEffect(() => {
    document.body.classList.toggle('dark', theme === 'dark');
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      <LocaleContext.Provider value={{ locale, setLocale }}>
        <ConfigProvider
          locale={getAntdLocale(locale)}
          theme={getAntdTheme(theme)}
        >
          <div
            style={{
              minHeight: '100vh',
              background: theme === 'dark' ? '#000' : '#f0f2f5',
            }}
          >
            {children}
          </div>
        </ConfigProvider>
      </LocaleContext.Provider>
    </ThemeContext.Provider>
  );
};

/**
 * 全局布局配置
 */
export const layout: RuntimeConfig['layout'] = () => {
  return {
    // 使用自定义布局
    pure: true,
    
    // 页面内容渲染
    childrenRender: (children: React.ReactNode) => {
      return (
        <RootWrapper>
          <LayoutContent>{children}</LayoutContent>
        </RootWrapper>
      );
    },

    // 页面切换时的处理
    onPageChange: () => {
      const token = localStorage.getItem('token');
      const { location } = window;
      // 如果没有登录，重定向到登录页
      if (!token && location.pathname !== '/user/login') {
        location.href = '/user/login';
      }
    },
  };
};

/**
 * 布局内容组件
 */
const LayoutContent: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { theme, setTheme } = React.useContext(ThemeContext);
  const { locale, setLocale } = React.useContext(LocaleContext);
  const [collapsed, setCollapsed] = useState(false);
  const [mobileMenuVisible, setMobileMenuVisible] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<string[]>(['dashboard']);
  const [openKeys, setOpenKeys] = useState<string[]>([]);
  
  // 响应式断点
  const screens = useBreakpoint();
  const isMobile = !screens.lg;

  // 获取当前路径对应的菜单项
  useEffect(() => {
    const path = window.location.pathname;
    const pathParts = path.split('/').filter(Boolean);
    if (pathParts.length > 0) {
      setSelectedKeys([pathParts[pathParts.length - 1]]);
    }
  }, []);

  // 处理菜单选择
  const handleMenuSelect = (keys: string[]) => {
    setSelectedKeys(keys);
    // 移动端选择后关闭菜单
    if (isMobile) {
      setMobileMenuVisible(false);
    }
  };

  // 处理子菜单展开
  const handleOpenChange = (keys: string[]) => {
    setOpenKeys(keys);
  };

  // 处理移动端菜单点击
  const handleMobileMenuClick = () => {
    setMobileMenuVisible(true);
  };

  // 计算内容区域边距
  const getContentMargin = () => {
    if (isMobile) {
      return 0;
    }
    return collapsed ? 80 : 240;
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      {/* Header */}
      <Header
        collapsed={collapsed}
        onCollapse={setCollapsed}
        theme={theme}
        onThemeChange={setTheme}
        locale={locale}
        onLocaleChange={setLocale}
        isMobile={isMobile}
        onMobileMenuClick={handleMobileMenuClick}
      />

      {/* 主体内容区域 */}
      <div style={{ display: 'flex', flex: 1, marginTop: 64 }}>
        {/* Sidebar */}
        <Sidebar
          collapsed={collapsed}
          theme={theme}
          selectedKeys={selectedKeys}
          openKeys={openKeys}
          onSelect={handleMenuSelect}
          onOpenChange={handleOpenChange}
          isMobile={isMobile}
          mobileVisible={mobileMenuVisible}
          onMobileClose={() => setMobileMenuVisible(false)}
          onCollapse={setCollapsed}
        />

        {/* 内容区域 */}
        <main
          style={{
            flex: 1,
            marginLeft: getContentMargin(),
            transition: 'margin-left 0.2s',
            minHeight: 'calc(100vh - 64px - 70px)',
            background: theme === 'dark' ? '#000' : '#f0f2f5',
            padding: isMobile ? '12px' : '24px',
            overflowX: 'auto',
          }}
        >
          <div style={{ 
            maxWidth: '100%',
            minWidth: isMobile ? 'auto' : 800,
          }}>
            {children}
          </div>
        </main>
      </div>

      {/* Footer */}
      {!isMobile && (
        <div
          style={{
            marginLeft: getContentMargin(),
            transition: 'margin-left 0.2s',
          }}
        >
          <Footer theme={theme} />
        </div>
      )}
    </div>
  );
};

/**
 * 请求配置
 */
export const request: RequestConfig = {
  timeout: 30000,
  // 请求拦截器
  requestInterceptors: [
    (config) => {
      const token = localStorage.getItem('token');
      if (token && config.headers) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      // 添加语言头
      const locale = getStoredLocale();
      if (config.headers) {
        config.headers['Accept-Language'] = locale;
      }
      return config;
    },
  ],
  // 响应拦截器
  responseInterceptors: [
    (response) => {
      const { data } = response;
      // 处理业务错误
      if (data && data.code !== 200 && data.code !== 0) {
        message.error(data.message || '请求失败');
        return Promise.reject(new Error(data.message));
      }
      return response;
    },
  ],
  // 错误处理
  errorConfig: {
    errorHandler: (error: any) => {
      if (error.response) {
        const { status } = error.response;
        switch (status) {
          case 401:
            notification.error({
              message: '未登录或登录已过期',
              description: '请重新登录',
            });
            localStorage.removeItem('token');
            window.location.href = '/user/login';
            break;
          case 403:
            notification.error({
              message: '没有权限',
              description: '您没有权限访问该资源',
            });
            break;
          case 500:
            notification.error({
              message: '服务器错误',
              description: '服务器发生错误，请稍后重试',
            });
            break;
          default:
            notification.error({
              message: `请求错误 ${status}`,
              description: error.response.data?.message || '未知错误',
            });
        }
      } else if (error.request) {
        notification.error({
          message: '网络错误',
          description: '无法连接到服务器，请检查网络',
        });
      } else {
        notification.error({
          message: '请求配置错误',
          description: error.message,
        });
      }
      throw error;
    },
  },
};

/**
 * 国际化文本
 */
export const i18n = {
  'zh-CN': {
    'app.title': 'MaaS-Router 管理后台',
    'app.welcome': '欢迎回来',
    'app.logout': '退出登录',
    'app.setting': '系统设置',
    'menu.dashboard': '仪表盘',
    'menu.api': 'API 管理',
    'menu.user': '用户管理',
    'menu.billing': '计费中心',
    'menu.analytics': '数据分析',
    'menu.system': '系统设置',
    'menu.redeem': '卡密管理',
    'menu.affiliate': '返利管理',
  },
  'en-US': {
    'app.title': 'MaaS-Router Admin',
    'app.welcome': 'Welcome back',
    'app.logout': 'Logout',
    'app.setting': 'Settings',
    'menu.dashboard': 'Dashboard',
    'menu.api': 'API Management',
    'menu.user': 'User Management',
    'menu.billing': 'Billing',
    'menu.analytics': 'Analytics',
    'menu.system': 'System Settings',
    'menu.redeem': 'Redeem Codes',
    'menu.affiliate': 'Affiliate',
  },
};

/**
 * 获取国际化文本
 */
export const t = (key: string, locale: LocaleType = 'zh-CN'): string => {
  return i18n[locale][key] || key;
};
