export default [
  {
    path: '/user',
    layout: false,
    routes: [
      { name: '登录', path: '/user/login', component: './User/Login' },
    ],
  },
  {
    path: '/welcome',
    name: '欢迎',
    icon: 'smile',
    component: './Welcome',
  },
  {
    path: '/dashboard',
    name: '仪表盘',
    icon: 'DashboardOutlined',
    component: './dashboard',
  },
  {
    path: '/users',
    name: '用户管理',
    icon: 'UserOutlined',
    routes: [
      { path: '/users', component: './users', name: '用户列表' },
      { path: '/users/:id', component: './users/detail', hideInMenu: true },
    ],
  },
  {
    path: '/api-keys',
    name: 'API Key管理',
    icon: 'KeyOutlined',
    component: './api-keys',
  },
  {
    path: '/providers',
    name: '供应商管理',
    icon: 'CloudOutlined',
    component: './providers',
  },
  {
    path: '/models',
    name: '模型管理',
    icon: 'RobotOutlined',
    component: './models',
  },
  {
    path: '/billing',
    name: '计费管理',
    icon: 'DollarOutlined',
    component: './billing',
  },
  {
    path: '/routing',
    name: '路由规则',
    icon: 'NodeIndexOutlined',
    component: './routing',
  },
  {
    path: '/monitoring',
    name: '监控告警',
    icon: 'AlertOutlined',
    component: './monitoring',
  },
  {
    path: '/balance',
    name: '账号余额',
    icon: 'WalletOutlined',
    component: './balance',
  },
  {
    path: '/channel-test',
    name: '渠道测试',
    icon: 'ApiOutlined',
    component: './channel-test',
  },
  {
    path: '/branding',
    name: '品牌设置',
    icon: 'SkinOutlined',
    component: './branding',
  },
  {
    path: '/system',
    name: '系统设置',
    icon: 'SettingOutlined',
    component: './system',
  },
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    component: './404',
  },
];
