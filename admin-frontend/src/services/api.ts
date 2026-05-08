import { request } from '@umijs/max';

// ==================== 通用类型定义 ====================

export interface ApiResponse<T = any> {
  code: number;
  data: T;
  message: string;
  success: boolean;
}

export interface PageParams {
  current?: number;
  pageSize?: number;
}

export interface PageResult<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

// ==================== 认证相关 ====================

export interface LoginParams {
  username: string;
  password: string;
  remember?: boolean;
}

export interface LoginResult {
  token: string;
  expires: number;
  user: UserInfo;
}

export async function login(params: LoginParams) {
  return request<ApiResponse<LoginResult>>('/api/v1/auth/login', {
    method: 'POST',
    data: params,
  });
}

export async function logout() {
  return request<ApiResponse<void>>('/api/v1/auth/logout', {
    method: 'POST',
  });
}

export async function getCurrentUser() {
  return request<ApiResponse<UserInfo>>('/api/v1/auth/current');
}

// ==================== 仪表盘 ====================

export interface DashboardStats {
  totalUsers: number;
  activeUsers: number;
  totalAccounts: number;
  activeAccounts: number;
  totalRequests: number;
  todayRequests: number;
  avgLatency: number;
  errorRate: number;
  totalRevenue: number;
  todayRevenue: number;
}

export interface RealtimeData {
  qps: number;
  activeConnections: number;
  latency: number;
  errorCount: number;
  timestamp: string;
}

export async function getDashboardStats() {
  return request<ApiResponse<DashboardStats>>('/api/v1/admin/dashboard/stats');
}

export async function getRealtimeData() {
  return request<ApiResponse<RealtimeData>>('/api/v1/admin/dashboard/realtime');
}

// ==================== 用户管理 ====================

export interface UserInfo {
  id: string;
  username: string;
  email: string;
  phone?: string;
  avatar?: string;
  status: 'active' | 'inactive' | 'banned';
  role: 'admin' | 'user' | 'viewer';
  balance: number;
  quotaLimit?: number;
  quotaUsed?: number;
  createdAt: string;
  updatedAt: string;
  lastLoginAt?: string;
}

export interface UserListParams extends PageParams {
  username?: string;
  email?: string;
  status?: string;
  role?: string;
}

export interface BalanceAdjustParams {
  amount: number;
  type: 'add' | 'subtract';
  description?: string;
}

export async function getUsers(params: UserListParams) {
  return request<ApiResponse<PageResult<UserInfo>>>('/api/v1/admin/users', {
    method: 'GET',
    params,
  });
}

export async function getUser(id: string) {
  return request<ApiResponse<UserInfo>>(`/api/v1/admin/users/${id}`);
}

export async function createUser(data: Partial<UserInfo> & { password: string }) {
  return request<ApiResponse<UserInfo>>('/api/v1/admin/users', {
    method: 'POST',
    data,
  });
}

export async function updateUser(id: string, data: Partial<UserInfo>) {
  return request<ApiResponse<UserInfo>>(`/api/v1/admin/users/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteUser(id: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/users/${id}`, {
    method: 'DELETE',
  });
}

export async function adjustUserBalance(id: string, data: BalanceAdjustParams) {
  return request<ApiResponse<UserInfo>>(`/api/v1/admin/users/${id}/balance`, {
    method: 'POST',
    data,
  });
}

// ==================== 账号管理 ====================

export interface AccountInfo {
  id: string;
  name: string;
  provider: 'openai' | 'anthropic' | 'azure' | 'google' | 'custom';
  apiKey: string;
  apiEndpoint?: string;
  status: 'active' | 'inactive' | 'error';
  priority: number;
  weight: number;
  models: string[];
  config: Record<string, any>;
  groupId?: string;
  groupName?: string;
  lastUsedAt?: string;
  testResult?: {
    success: boolean;
    latency: number;
    error?: string;
    testedAt: string;
  };
  createdAt: string;
  updatedAt: string;
}

export interface AccountListParams extends PageParams {
  name?: string;
  provider?: string;
  status?: string;
  groupId?: string;
}

export async function getAccounts(params: AccountListParams) {
  return request<ApiResponse<PageResult<AccountInfo>>>('/api/v1/admin/accounts', {
    method: 'GET',
    params,
  });
}

export async function getAccount(id: string) {
  return request<ApiResponse<AccountInfo>>(`/api/v1/admin/accounts/${id}`);
}

export async function createAccount(data: Partial<AccountInfo>) {
  return request<ApiResponse<AccountInfo>>('/api/v1/admin/accounts', {
    method: 'POST',
    data,
  });
}

export async function updateAccount(id: string, data: Partial<AccountInfo>) {
  return request<ApiResponse<AccountInfo>>(`/api/v1/admin/accounts/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteAccount(id: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/accounts/${id}`, {
    method: 'DELETE',
  });
}

export async function testAccount(id: string) {
  return request<ApiResponse<{ success: boolean; latency: number; error?: string }>>(`/api/v1/admin/accounts/${id}/test`, {
    method: 'POST',
  });
}

export async function refreshAccountToken(id: string) {
  return request<ApiResponse<AccountInfo>>(`/api/v1/admin/accounts/${id}/refresh`, {
    method: 'POST',
  });
}

// ==================== 分组管理 ====================

export interface GroupInfo {
  id: string;
  name: string;
  description?: string;
  status: 'active' | 'inactive';
  priority: number;
  strategy: 'round-robin' | 'weighted' | 'priority' | 'random';
  accountCount: number;
  accounts?: AccountInfo[];
  createdAt: string;
  updatedAt: string;
}

export interface GroupListParams extends PageParams {
  name?: string;
  status?: string;
}

export async function getGroups(params: GroupListParams) {
  return request<ApiResponse<PageResult<GroupInfo>>>('/api/v1/admin/groups', {
    method: 'GET',
    params,
  });
}

export async function getGroup(id: string) {
  return request<ApiResponse<GroupInfo>>(`/api/v1/admin/groups/${id}`);
}

export async function createGroup(data: Partial<GroupInfo>) {
  return request<ApiResponse<GroupInfo>>('/api/v1/admin/groups', {
    method: 'POST',
    data,
  });
}

export async function updateGroup(id: string, data: Partial<GroupInfo>) {
  return request<ApiResponse<GroupInfo>>(`/api/v1/admin/groups/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteGroup(id: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/groups/${id}`, {
    method: 'DELETE',
  });
}

export async function addAccountToGroup(groupId: string, accountId: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/groups/${groupId}/accounts`, {
    method: 'POST',
    data: { accountId },
  });
}

export async function removeAccountFromGroup(groupId: string, accountId: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/groups/${groupId}/accounts/${accountId}`, {
    method: 'DELETE',
  });
}

// ==================== 路由规则 ====================

export interface RoutingRule {
  id: string;
  name: string;
  description?: string;
  priority: number;
  status: 'active' | 'inactive';
  conditions: {
    field: string;
    operator: string;
    value: string;
  }[];
  action: {
    type: 'group' | 'account' | 'model' | 'fallback';
    target: string;
  };
  createdAt: string;
  updatedAt: string;
}

export interface RoutingRuleListParams extends PageParams {
  name?: string;
  status?: string;
}

export async function getRoutingRules(params?: RoutingRuleListParams) {
  return request<ApiResponse<PageResult<RoutingRule>>>('/api/v1/admin/router-rules', {
    method: 'GET',
    params,
  });
}

export async function createRoutingRule(data: Partial<RoutingRule>) {
  return request<ApiResponse<RoutingRule>>('/api/v1/admin/router-rules', {
    method: 'POST',
    data,
  });
}

export async function updateRoutingRule(id: string, data: Partial<RoutingRule>) {
  return request<ApiResponse<RoutingRule>>(`/api/v1/admin/router-rules/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteRoutingRule(id: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/router-rules/${id}`, {
    method: 'DELETE',
  });
}

// ==================== 运维监控 ====================

export interface ConcurrencyStats {
  total: number;
  byProvider: { provider: string; count: number }[];
  byModel: { model: string; count: number }[];
  byUser: { userId: string; username: string; count: number }[];
}

export interface RealtimeTraffic {
  timestamp: string;
  qps: number;
  latency: number;
  errorRate: number;
  bandwidth: number;
}

export interface ErrorLog {
  id: string;
  timestamp: string;
  level: 'error' | 'warning' | 'critical';
  type: string;
  message: string;
  details?: string;
  accountId?: string;
  accountName?: string;
  userId?: string;
  username?: string;
  requestId?: string;
}

export interface ErrorLogListParams extends PageParams {
  level?: string;
  type?: string;
  startTime?: string;
  endTime?: string;
}

export interface AlertRule {
  id: string;
  name: string;
  metric: string;
  operator: 'gt' | 'lt' | 'eq' | 'gte' | 'lte';
  threshold: number;
  duration: number;
  status: 'active' | 'inactive';
  channels: string[];
  createdAt: string;
}

export async function getConcurrencyStats() {
  return request<ApiResponse<ConcurrencyStats>>('/api/v1/admin/ops/concurrency');
}

export async function getRealtimeTraffic(params?: { duration?: number }) {
  return request<ApiResponse<RealtimeTraffic[]>>('/api/v1/admin/ops/realtime-traffic', {
    method: 'GET',
    params,
  });
}

export async function getErrorLogs(params: ErrorLogListParams) {
  return request<ApiResponse<PageResult<ErrorLog>>>('/api/v1/admin/ops/errors', {
    method: 'GET',
    params,
  });
}

export async function getAlertRules(params?: PageParams) {
  return request<ApiResponse<PageResult<AlertRule>>>('/api/v1/admin/ops/alert-rules', {
    method: 'GET',
    params,
  });
}

// ==================== API Key 管理 ====================

export interface ApiKeyInfo {
  id: string;
  name: string;
  key: string;
  keyPrefix: string;
  userId: string;
  status: 'active' | 'inactive' | 'revoked';
  permissions: string[];
  rateLimit?: number;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
}

export async function getApiKeys(params: PageParams & { userId?: string }) {
  return request<ApiResponse<PageResult<ApiKeyInfo>>>('/api/v1/admin/api-keys', {
    method: 'GET',
    params,
  });
}

export async function createApiKey(data: Partial<ApiKeyInfo>) {
  return request<ApiResponse<ApiKeyInfo>>('/api/v1/admin/api-keys', {
    method: 'POST',
    data,
  });
}

export async function revokeApiKey(id: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/api-keys/${id}/revoke`, {
    method: 'POST',
  });
}

// ==================== 模型管理 ====================

export interface ModelInfo {
  id: string;
  name: string;
  providerId: string;
  providerName?: string;
  type: 'chat' | 'completion' | 'embedding';
  status: 'active' | 'inactive';
  pricing: {
    inputPrice: number;
    outputPrice: number;
    currency: string;
  };
  capabilities: string[];
  contextWindow: number;
  config: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export async function getModels(params?: PageParams & { providerId?: string }) {
  return request<ApiResponse<PageResult<ModelInfo>>>('/api/v1/admin/models', {
    method: 'GET',
    params,
  });
}

export async function createModel(data: Partial<ModelInfo>) {
  return request<ApiResponse<ModelInfo>>('/api/v1/admin/models', {
    method: 'POST',
    data,
  });
}

export async function updateModel(id: string, data: Partial<ModelInfo>) {
  return request<ApiResponse<ModelInfo>>(`/api/v1/admin/models/${id}`, {
    method: 'PUT',
    data,
  });
}

export async function deleteModel(id: string) {
  return request<ApiResponse<void>>(`/api/v1/admin/models/${id}`, {
    method: 'DELETE',
  });
}

// ==================== 计费管理 ====================

export interface BillingRecord {
  id: string;
  userId: string;
  username?: string;
  type: 'charge' | 'consumption' | 'refund';
  amount: number;
  balance: number;
  description: string;
  relatedId?: string;
  createdAt: string;
}

export interface BillingStats {
  totalRevenue: number;
  totalConsumption: number;
  activeUsers: number;
  todayRevenue: number;
}

export async function getBillingRecords(params: PageParams & { userId?: string; type?: string }) {
  return request<ApiResponse<PageResult<BillingRecord>>>('/api/v1/admin/billing/records', {
    method: 'GET',
    params,
  });
}

export async function getBillingStats() {
  return request<ApiResponse<BillingStats>>('/api/v1/admin/billing/stats');
}

// ==================== 系统设置 ====================

export interface SystemConfig {
  id: string;
  key: string;
  value: any;
  description?: string;
  category: string;
  updatedAt: string;
}

export async function getSystemConfigs(params?: { category?: string }) {
  return request<ApiResponse<SystemConfig[]>>('/api/v1/admin/system/configs', {
    method: 'GET',
    params,
  });
}

export async function updateSystemConfig(key: string, value: any) {
  return request<ApiResponse<SystemConfig>>(`/api/v1/admin/system/configs/${key}`, {
    method: 'PUT',
    data: { value },
  });
}

export async function getSystemStatus() {
  return request<ApiResponse<{
    version: string;
    uptime: string;
    database: 'connected' | 'disconnected';
    cache: 'connected' | 'disconnected';
    memory: { used: number; total: number };
    cpu: { usage: number };
  }>>('/api/v1/admin/system/status');
}

// ==================== 账号余额 ====================

export interface BalanceInfo {
  id: string;
  accountName: string;
  platform: string;
  balance: number;
  currency: string;
  usedToday: number;
  lastUpdated: string;
  status: 'active' | 'inactive' | 'error';
}

export async function getAccountBalance(accountId: string) {
  return request<ApiResponse<BalanceInfo>>(`/api/v1/admin/accounts/${accountId}/balance`);
}

export async function getAllBalances() {
  return request<ApiResponse<BalanceInfo[]>>('/api/v1/admin/accounts/balances');
}

export async function refreshBalance(accountId?: string) {
  const url = accountId
    ? `/api/v1/admin/accounts/${accountId}/balance/refresh`
    : '/api/v1/admin/accounts/balances/refresh';
  return request<ApiResponse<BalanceInfo[]>>(url, {
    method: 'POST',
  });
}

// ==================== 渠道测试 ====================

export interface ChannelTestResult {
  id: string;
  accountName: string;
  platform: string;
  status: 'healthy' | 'unhealthy' | 'unknown';
  latency: number;
  lastTest: string;
  error: string;
}

export async function testAccountChannel(accountId: string) {
  return request<ApiResponse<ChannelTestResult>>(`/api/v1/admin/accounts/${accountId}/test`, {
    method: 'POST',
  });
}

export async function testAllAccounts() {
  return request<ApiResponse<ChannelTestResult[]>>('/api/v1/admin/accounts/test-all', {
    method: 'POST',
  });
}

export async function getTestResults() {
  return request<ApiResponse<ChannelTestResult[]>>('/api/v1/admin/accounts/test-results');
}

// ==================== 品牌设置 ====================

export interface BrandingSettings {
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

export async function getBrandingSettings() {
  return request<ApiResponse<BrandingSettings>>('/api/v1/admin/branding');
}

export async function updateBrandingSettings(data: Partial<BrandingSettings>) {
  return request<ApiResponse<BrandingSettings>>('/api/v1/admin/branding', {
    method: 'PUT',
    data,
  });
}
