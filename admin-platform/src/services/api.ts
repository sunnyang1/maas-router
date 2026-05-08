const BASE_URL = "/api/admin/v1";

let authToken: string | null = localStorage.getItem("admin_token");

export function setAuthToken(token: string | null) {
  authToken = token;
  if (token) localStorage.setItem("admin_token", token);
  else localStorage.removeItem("admin_token");
}

export function getAuthToken() {
  return authToken;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options?.headers as Record<string, string>) || {}),
  };

  if (authToken) {
    headers["Authorization"] = `Bearer ${authToken}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
  });

  if (res.status === 401) {
    setAuthToken(null);
    window.location.href = "/login";
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: res.statusText }));
    throw new Error(err.detail || "Request failed");
  }

  return res.json();
}

// ============================================
// Auth
// ============================================
export const authApi = {
  login: (email: string, password: string) =>
    request<{ access_token: string; user: any }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  me: () => request<any>("/auth/me"),
};

// ============================================
// Dashboard
// ============================================
export const dashboardApi = {
  overview: () => request<any>("/dashboard/overview"),
  trends: (days = 7) => request<any>(`/dashboard/trends?days=${days}`),
  modelDistribution: () => request<any>("/dashboard/model-distribution"),
  recentRequests: (limit = 10) =>
    request<any>(`/dashboard/recent-requests?limit=${limit}`),
};

// ============================================
// Users
// ============================================
export const usersApi = {
  list: (params?: Record<string, any>) => {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return request<any>(`/users${qs}`);
  },
  get: (id: string) => request<any>(`/users/${id}`),
  create: (data: any) =>
    request<any>("/users", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: any) =>
    request<any>(`/users/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  getApiKeys: (id: string) => request<any>(`/users/${id}/api-keys`),
  getTransactions: (id: string, params?: Record<string, any>) => {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return request<any>(`/users/${id}/transactions${qs}`);
  },
};

// ============================================
// Models & Providers
// ============================================
export const modelsApi = {
  listProviders: () => request<any>("/providers"),
  updateProvider: (id: string, data: any) =>
    request<any>(`/providers/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  toggleProvider: (id: string, status: string) =>
    request<any>(`/providers/${id}/status?status=${status}`, { method: "PUT" }),

  listModels: (params?: Record<string, any>) => {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return request<any>(`/models${qs}`);
  },
  updateModel: (id: string, data: any) =>
    request<any>(`/models/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  toggleModel: (id: string, status: string) =>
    request<any>(`/models/${id}/status?status=${status}`, { method: "PUT" }),

  listRoutingRules: () => request<any>("/routing-rules"),
  createRoutingRule: (data: any) =>
    request<any>("/routing-rules", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  deleteRoutingRule: (id: string) =>
    request<any>(`/routing-rules/${id}`, { method: "DELETE" }),
};

// ============================================
// Billing
// ============================================
export const billingApi = {
  overview: () => request<any>("/billing/overview"),
  transactions: (params?: Record<string, any>) => {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return request<any>(`/billing/transactions${qs}`);
  },
  adjustBalance: (data: any) =>
    request<any>("/billing/adjust", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  credSupply: () => request<any>("/cred/supply"),
};

// ============================================
// Monitoring
// ============================================
export const monitoringApi = {
  services: () => request<any>("/monitoring/services"),
  metrics: () => request<any>("/monitoring/metrics"),
  failoverLogs: () => request<any>("/monitoring/failover-logs"),
  alerts: () => request<any>("/monitoring/alerts"),
};

// ============================================
// Settings
// ============================================
export const settingsApi = {
  getConfig: () => request<any>("/system/config"),
  updateConfig: (key: string, value: any) =>
    request<any>(`/system/config/${key}`, {
      method: "PUT",
      body: JSON.stringify(value),
    }),
  listAdmins: () => request<any>("/system/admins"),
  createAdmin: (data: any) =>
    request<any>("/system/admins", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  auditLogs: (params?: Record<string, any>) => {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return request<any>(`/audit-logs${qs}`);
  },
};
