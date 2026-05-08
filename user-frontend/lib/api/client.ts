import axios, { AxiosInstance, AxiosError } from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.maas-router.com';

// Token 管理类型
interface TokenResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

interface AuthResponse extends TokenResponse {
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

  constructor() {
    this.client = axios.create({
      baseURL: `${API_BASE_URL}/api/v1`,
      headers: {
        'Content-Type': 'application/json',
      },
      timeout: 30000,
    });

    // Request interceptor - 添加 JWT Token
    this.client.interceptors.request.use(
      (config) => {
        const accessToken = this.getAccessToken();
        if (accessToken) {
          config.headers.Authorization = `Bearer ${accessToken}`;
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
            this.clearTokens();
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

            // 使用新的 Token 重试原请求
            const accessToken = this.getAccessToken();
            if (accessToken) {
              originalRequest.headers.Authorization = `Bearer ${accessToken}`;
            }
            return this.client(originalRequest);
          } catch (refreshError) {
            this.refreshPromise = null;
            this.clearTokens();
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

  // Token 管理
  private getAccessToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('access-token');
  }

  private getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('refresh-token');
  }

  private setTokens(accessToken: string, refreshToken: string) {
    if (typeof window === 'undefined') return;
    localStorage.setItem('access-token', accessToken);
    localStorage.setItem('refresh-token', refreshToken);
  }

  clearTokens() {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('access-token');
    localStorage.removeItem('refresh-token');
    localStorage.removeItem('api-key');
  }

  private async refreshAccessToken(): Promise<void> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await axios.post<TokenResponse>(
      `${API_BASE_URL}/api/v1/auth/refresh`,
      { refreshToken },
      { headers: { 'Content-Type': 'application/json' } }
    );

    this.setTokens(response.data.accessToken, response.data.refreshToken);
  }

  // ==================== 认证 API ====================

  async register(email: string, password: string, name: string): Promise<AuthResponse> {
    const response = await this.client.post<AuthResponse>('/auth/register', { 
      email, 
      password, 
      name 
    });
    const { accessToken, refreshToken } = response.data;
    this.setTokens(accessToken, refreshToken);
    return response.data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await this.client.post<AuthResponse>('/auth/login', { 
      email, 
      password 
    });
    const { accessToken, refreshToken } = response.data;
    this.setTokens(accessToken, refreshToken);
    return response.data;
  }

  async logout(): Promise<void> {
    try {
      await this.client.post('/auth/logout');
    } finally {
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
    const key = apiKey || localStorage.getItem('api-key') || '';
    
    const response = await fetch(`${API_BASE_URL}/v1/chat/completions`, {
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
    const key = apiKey || localStorage.getItem('api-key') || '';
    
    const response = await fetch(`${API_BASE_URL}/v1/chat/completions`, {
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
    const key = apiKey || localStorage.getItem('api-key') || '';
    
    const response = await fetch(`${API_BASE_URL}/v1/models`, {
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
    const key = apiKey || localStorage.getItem('api-key') || '';
    
    const response = await fetch(`${API_BASE_URL}/v1/embeddings`, {
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
    const key = apiKey || localStorage.getItem('api-key') || '';
    
    const response = await fetch(`${API_BASE_URL}/v1/images/generations`, {
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
    const key = apiKey || localStorage.getItem('api-key') || '';
    
    const response = await fetch(`${API_BASE_URL}/v1/messages`, {
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
