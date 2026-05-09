import axios, { AxiosInstance, AxiosError } from 'axios';
import { getApiBaseUrl, getRuntimeConfig, initRuntimeConfig } from './runtime-config';

// 认证响应类型 (Token 现在存储在 httpOnly cookie 中)
interface AuthResponse {
  user: {
    id: string;
    email: string;
    name: string;
    tier: 'free' | 'pro' | 'enterprise';
    status: 'active' | 'suspended';
    credBalance: number;
  };
}

// API Key 类型
export interface ApiKey {
  id: string;
  name: string;
  keyPrefix: string;
  status: 'active' | 'disabled';
  dailyLimit?: number;
  monthlyLimit?: number;
  lastUsedAt?: string;
  createdAt: string;
  expiresAt?: string;
}

export interface CreateApiKeyRequest {
  name: string;
  dailyLimit?: number;
  monthlyLimit?: number;
  expiresAt?: string;
}

// 使用记录类型
export interface UsageRecord {
  id: string;
  apiKeyId: string;
  apiKeyName: string;
  model: string;
  provider: string;
  requestType: 'chat' | 'embedding' | 'image';
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  cost: number;
  latency: number;
  status: 'success' | 'error';
  errorMessage?: string;
  createdAt: string;
}

export interface UsageStats {
  totalRequests: number;
  totalTokens: number;
  totalCost: number;
  avgLatency: number;
  successRate: number;
  byModel: Array<{
    model: string;
    requests: number;
    tokens: number;
    cost: number;
  }>;
  byProvider: Array<{
    provider: string;
    requests: number;
    tokens: number;
    cost: number;
  }>;
}

export interface DashboardData {
  todayCost: number;
  weekCost: number;
  monthCost: number;
  costTrend?: number;
  weekTrend?: number;
  monthTrend?: number;
  requestHistory: Array<{
    date: string;
    requests: number;
    cost: number;
  }>;
  routerDistribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;
  recentRequests: UsageRecord[];
}

class ApiClient {
  public client: AxiosInstance;
  private refreshPromise: Promise<void> | null = null;
  private baseUrl: string = '';
  private initialized: boolean = false;

  constructor() {
    // 初始化时使用环境变量或空字符串
    this.baseUrl = process.env.NEXT_PUBLIC_API_URL || '';

    this.client = axios.create({
      baseURL: this.getBaseUrl(),
      headers: {
        'Content-Type': 'application/json',
      },
      timeout: 30000,
      maxContentLength: 10 * 1024 * 1024, // 10MB
      maxBodyLength: 10 * 1024 * 1024,    // 10MB
      withCredentials: true, // 允许携带 cookies (包括 httpOnly cookies)
    });

    this.setupInterceptors();
  }

  /**
   * 获取 API 基础 URL
   * 优先使用运行时配置，其次使用构建时环境变量
   */
  private getBaseUrl(): string {
    const apiUrl = getApiBaseUrl();
    return apiUrl ? `${apiUrl}/api/v1` : '/api/v1';
  }

  /**
   * 初始化运行时配置
   * 应在应用启动时调用
   */
  public async init(): Promise<void> {
    if (this.initialized) return;

    // 加载运行时配置
    await getRuntimeConfig();

    // 更新 baseURL
    const newBaseUrl = this.getBaseUrl();
    if (newBaseUrl !== this.client.defaults.baseURL) {
      this.client.defaults.baseURL = newBaseUrl;
    }

    this.initialized = true;
  }

  /**
   * 设置请求和响应拦截器
   */
  private setupInterceptors(): void {
    // Request interceptor - 确保配置已初始化
    this.client.interceptors.request.use(
      async (config) => {
        // 确保配置已初始化
        if (!this.initialized && typeof window !== 'undefined') {
          await this.init();
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor - 处理 Token 过期和自动刷新
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as any;

        // 如果是 401 错误且不是刷新 Token 的请求
        if (error.response?.status === 401 && !originalRequest._retry) {
          // 如果是刷新 Token 失败，直接登出
          if (originalRequest.url?.includes('/auth/refresh')) {
            if (typeof window !== 'undefined') {
              window.location.href = '/login';
            }
            return Promise.reject(error);
          }

          originalRequest._retry = true;

          try {
            // 等待 Token 刷新完成
            if (!this.refreshPromise) {
              this.refreshPromise = this.refreshAccessToken();
            }
            await this.refreshPromise;
            this.refreshPromise = null;

            // 重试原请求 (cookie 会自动携带)
            return this.client(originalRequest);
          } catch (refreshError) {
            this.refreshPromise = null;
            if (typeof window !== 'undefined') {
              window.location.href = '/login';
            }
            return Promise.reject(refreshError);
          }
        }

        return Promise.reject(error);
      }
    );
  }

  // 清除 API Key (Token 现在存储在 httpOnly cookie 中，由后端管理)
  clearTokens() {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('api-key');
  }

  private async refreshAccessToken(): Promise<void> {
    const baseUrl = getApiBaseUrl();
    const apiBaseUrl = baseUrl ? `${baseUrl}/api/v1` : '/api/v1';

    // 使用 withCredentials 发送请求，后端会从 cookie 中读取 refresh token
    await axios.post(
      `${apiBaseUrl}/auth/refresh`,
      {},
      {
        headers: { 'Content-Type': 'application/json' },
        withCredentials: true,
      }
    );
    // 新的 access token 会通过 httpOnly cookie 自动设置
  }

  /**
   * 获取网关 API 基础 URL（用于流式请求）
   */
  private getGatewayBaseUrl(): string {
    const baseUrl = getApiBaseUrl();
    return baseUrl || '';
  }

  // ==================== 认证 API ====================

  async register(email: string, password: string, name: string): Promise<AuthResponse> {
    const response = await this.client.post<AuthResponse>('/auth/register', {
      email,
      password,
      name
    });
    // Token 已通过 httpOnly cookie 自动设置，无需手动存储
    return response.data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await this.client.post<AuthResponse>('/auth/login', {
      email,
      password
    });
    // Token 已通过 httpOnly cookie 自动设置，无需手动存储
    return response.data;
  }

  async logout(): Promise<void> {
    try {
      await this.client.post('/auth/logout');
    } finally {
      // Cookie 已由后端清除，前端只需清理本地状态
      this.clearTokens();
    }
  }

  async forgotPassword(email: string): Promise<{ message: string }> {
    const response = await this.client.post('/auth/forgot-password', { email });
    return response.data;
  }

  async resetPassword(token: string, password: string): Promise<{ message: string }> {
    const response = await this.client.post('/auth/reset-password', {
      token,
      password
    });
    return response.data;
  }

  // ==================== 用户 API ====================

  async getProfile(): Promise<{
    id: string;
    email: string;
    name: string;
    tier: 'free' | 'pro' | 'enterprise';
    status: 'active' | 'suspended';
    credBalance: number;
    createdAt: string;
  }> {
    const response = await this.client.get('/user/profile');
    return response.data;
  }

  async updateProfile(data: { name?: string; email?: string }): Promise<{
    id: string;
    email: string;
    name: string;
  }> {
    const response = await this.client.put('/user/profile', data);
    return response.data;
  }

  async changePassword(data: {
    currentPassword: string;
    newPassword: string
  }): Promise<{ message: string }> {
    const response = await this.client.put('/user/password', data);
    return response.data;
  }

  // ==================== API Key 管理 ====================

  async listApiKeys(): Promise<ApiKey[]> {
    const response = await this.client.get<ApiKey[]>('/keys');
    return response.data;
  }

  async getApiKey(id: string): Promise<ApiKey & { fullKey?: string }> {
    const response = await this.client.get(`/keys/${id}`);
    return response.data;
  }

  async createApiKey(data: CreateApiKeyRequest): Promise<ApiKey & { key: string }> {
    const response = await this.client.post('/keys', data);
    return response.data;
  }

  async updateApiKey(id: string, data: Partial<CreateApiKeyRequest & { status?: 'active' | 'disabled' }>): Promise<ApiKey> {
    const response = await this.client.put(`/keys/${id}`, data);
    return response.data;
  }

  async deleteApiKey(id: string): Promise<void> {
    await this.client.delete(`/keys/${id}`);
  }

  // ==================== 使用记录 ====================

  async getUsageList(params?: {
    page?: number;
    limit?: number;
    apiKeyId?: string;
    model?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{
    data: UsageRecord[];
    pagination: {
      page: number;
      limit: number;
      total: number;
      totalPages: number;
    };
  }> {
    const response = await this.client.get('/usage', { params });
    return response.data;
  }

  async getUsageStats(params?: {
    startDate?: string;
    endDate?: string;
    apiKeyId?: string;
  }): Promise<UsageStats> {
    const response = await this.client.get('/usage/stats', { params });
    return response.data;
  }

  async getDashboard(): Promise<DashboardData> {
    const response = await this.client.get('/usage/dashboard');
    return response.data;
  }

  // 兼容旧 API 名称
  async getDashboardStats(): Promise<DashboardData> {
    return this.getDashboard();
  }

  async getUsageHistory(period: 'day' | 'week' | 'month'): Promise<{
    data: UsageRecord[];
  }> {
    const response = await this.client.get('/usage', {
      params: { period }
    });
    return response.data;
  }

  // ==================== 网关 API (需 API Key) ====================

  async *streamChat(data: {
    model: string;
    messages: Array<{ role: string; content: string }>;
    stream?: boolean;
    temperature?: number;
    max_tokens?: number;
  }, apiKey?: string): AsyncGenerator<any, void, unknown> {
    const key = apiKey || (typeof window !== 'undefined' ? localStorage.getItem('api-key') : '') || '';
    const baseUrl = this.getGatewayBaseUrl();

    const response = await fetch(`${baseUrl}/v1/chat/completions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key}`,
      },
      body: JSON.stringify({ ...data, stream: true }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.error?.message || `HTTP error! status: ${response.status}`);
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error('No response body');

    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data === '[DONE]') return;
          try {
            yield JSON.parse(data);
          } catch {
            // Ignore parse errors
          }
        }
      }
    }
  }

  async chatCompletion(data: {
    model: string;
    messages: Array<{ role: string; content: string }>;
    temperature?: number;
    max_tokens?: number;
  }, apiKey?: string): Promise<any> {
    const key = apiKey || (typeof window !== 'undefined' ? localStorage.getItem('api-key') : '') || '';
    const baseUrl = this.getGatewayBaseUrl();

    const response = await fetch(`${baseUrl}/v1/chat/completions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key}`,
      },
      body: JSON.stringify({ ...data, stream: false }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.error?.message || `HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async getModels(apiKey?: string): Promise<{
    data: Array<{
      id: string;
      object: string;
      owned_by: string;
    }>;
  }> {
    const key = apiKey || (typeof window !== 'undefined' ? localStorage.getItem('api-key') : '') || '';
    const baseUrl = this.getGatewayBaseUrl();

    const response = await fetch(`${baseUrl}/v1/models`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${key}`,
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async createEmbedding(data: {
    model: string;
    input: string | string[];
  }, apiKey?: string): Promise<any> {
    const key = apiKey || (typeof window !== 'undefined' ? localStorage.getItem('api-key') : '') || '';
    const baseUrl = this.getGatewayBaseUrl();

    const response = await fetch(`${baseUrl}/v1/embeddings`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.error?.message || `HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  async createImage(data: {
    model: string;
    prompt: string;
    n?: number;
    size?: string;
  }, apiKey?: string): Promise<any> {
    const key = apiKey || (typeof window !== 'undefined' ? localStorage.getItem('api-key') : '') || '';
    const baseUrl = this.getGatewayBaseUrl();

    const response = await fetch(`${baseUrl}/v1/images/generations`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.error?.message || `HTTP error! status: ${response.status}`);
    }

    return response.json();
  }

  // Claude Messages API
  async *streamClaudeMessages(data: {
    model: string;
    messages: Array<{ role: string; content: string }>;
    max_tokens: number;
    system?: string;
  }, apiKey?: string): AsyncGenerator<any, void, unknown> {
    const key = apiKey || (typeof window !== 'undefined' ? localStorage.getItem('api-key') : '') || '';
    const baseUrl = this.getGatewayBaseUrl();

    const response = await fetch(`${baseUrl}/v1/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': key,
        'anthropic-version': '2023-06-01',
      },
      body: JSON.stringify({ ...data, stream: true }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.error?.message || `HTTP error! status: ${response.status}`);
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error('No response body');

    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          try {
            yield JSON.parse(data);
          } catch {
            // Ignore parse errors
          }
        }
      }
    }
  }
}

export const apiClient = new ApiClient();

// 导出初始化函数，供应用启动时调用
export const initApiClient = () => apiClient.init();

// 导出运行时配置相关函数
export { initRuntimeConfig, getRuntimeConfig, getApiBaseUrl } from './runtime-config';
