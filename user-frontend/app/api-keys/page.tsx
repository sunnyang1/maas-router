'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Plus, Trash2, Eye, EyeOff, Copy, MoreVertical,
  Key, Loader2, X, Check, AlertTriangle, Clock,
  ToggleLeft, ToggleRight, Activity, Coins, Zap,
  Shield, Ban, CheckCircle2, AlertCircle, ChevronDown,
  BarChart3, Globe, Hash
} from 'lucide-react';
import { apiClient, ApiKey, CreateApiKeyRequest } from '@/lib/api/client';
import { DashboardHeader } from '@/components/dashboard/DashboardHeader';

// 可选模型列表
const AVAILABLE_MODELS = [
  { id: 'deepseek-v4-flash', name: 'DeepSeek-V4-Flash', tier: 'economy' },
  { id: 'deepseek-v4-pro', name: 'DeepSeek-V4-Pro', tier: 'standard' },
  { id: 'claude-sonnet-4', name: 'Claude Sonnet 4', tier: 'standard' },
  { id: 'claude-opus-4', name: 'Claude Opus 4', tier: 'premium' },
  { id: 'gpt-4.1', name: 'GPT-4.1', tier: 'standard' },
  { id: 'gpt-4.1-mini', name: 'GPT-4.1 Mini', tier: 'economy' },
];

// 密钥使用统计类型
interface KeyUsageStats {
  monthRequests: number;
  monthTokens: number;
  monthCost: number;
}

// 健康状态类型
type HealthStatus = 'healthy' | 'warning' | 'critical';

function getHealthStatus(key: ApiKey, stats?: KeyUsageStats): HealthStatus {
  if (key.status === 'disabled') return 'critical';
  if (!stats) return 'healthy';

  const usageRatio = key.monthlyLimit
    ? stats.monthRequests / key.monthlyLimit
    : key.dailyLimit
    ? stats.monthRequests / (key.dailyLimit * 30)
    : 0;

  if (usageRatio >= 1) return 'critical';
  if (usageRatio >= 0.8) return 'warning';
  return 'healthy';
}

const HEALTH_CONFIG: Record<HealthStatus, {
  label: string;
  color: string;
  bgColor: string;
  dotColor: string;
  icon: typeof CheckCircle2;
}> = {
  healthy: {
    label: '正常',
    color: 'text-success',
    bgColor: 'bg-success/10',
    dotColor: 'bg-success',
    icon: CheckCircle2,
  },
  warning: {
    label: '接近限额',
    color: 'text-[#f59e0b]',
    bgColor: 'bg-[#f59e0b]/10',
    dotColor: 'bg-[#f59e0b]',
    icon: AlertCircle,
  },
  critical: {
    label: '已超限/禁用',
    color: 'text-error',
    bgColor: 'bg-error/10',
    dotColor: 'bg-error',
    icon: AlertTriangle,
  },
};

export default function ApiKeysPage() {
  const queryClient = useQueryClient();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newKeyData, setNewKeyData] = useState<CreateApiKeyRequest & {
    allowedModels?: string[];
    ipWhitelist?: string;
    rpmLimit?: number;
  }>({ name: '' });
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copiedKeyId, setCopiedKeyId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [isModelSelectOpen, setIsModelSelectOpen] = useState(false);

  // 获取 API Keys 列表
  const { data: apiKeys, isLoading } = useQuery({
    queryKey: ['api-keys'],
    queryFn: () => apiClient.listApiKeys(),
  });

  // 获取密钥使用统计（优雅降级）
  const { data: keyUsageMap } = useQuery({
    queryKey: ['key-usage-stats'],
    queryFn: async () => {
      try {
        const response = await apiClient.client.get<Record<string, KeyUsageStats>>('/keys/usage-stats');
        return response.data;
      } catch {
        return {};
      }
    },
    retry: 1,
    staleTime: 60000,
  });

  // 创建 API Key
  const createMutation = useMutation({
    mutationFn: (data: CreateApiKeyRequest) => apiClient.createApiKey(data),
    onSuccess: (data) => {
      setCreatedKey(data.key);
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });

  // 删除 API Key
  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiClient.deleteApiKey(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
      setDeleteConfirmId(null);
      setSelectedKeys((prev) => {
        const next = new Set(prev);
        next.clear();
        return next;
      });
    },
  });

  // 更新 API Key 状态
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { status: 'active' | 'disabled' } }) =>
      apiClient.updateApiKey(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });

  // 批量更新状态
  const batchUpdateMutation = useMutation({
    mutationFn: async ({ ids, status }: { ids: string[]; status: 'active' | 'disabled' }) => {
      await Promise.all(
        ids.map((id) => apiClient.updateApiKey(id, { status }))
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
      setSelectedKeys(new Set());
    },
  });

  const handleCreateKey = () => {
    if (!newKeyData.name.trim()) return;
    createMutation.mutate(newKeyData);
  };

  const handleCopyKey = async (key: string, id: string) => {
    await navigator.clipboard.writeText(key);
    setCopiedKeyId(id);
    setTimeout(() => setCopiedKeyId(null), 2000);
  };

  const handleCloseCreateModal = () => {
    setIsCreateModalOpen(false);
    setNewKeyData({ name: '' });
    setCreatedKey(null);
    createMutation.reset();
  };

  const toggleKeySelection = (keyId: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(keyId)) {
        next.delete(keyId);
      } else {
        next.add(keyId);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (!apiKeys) return;
    if (selectedKeys.size === apiKeys.length) {
      setSelectedKeys(new Set());
    } else {
      setSelectedKeys(new Set(apiKeys.map((k) => k.id)));
    }
  };

  const toggleModelSelection = (modelId: string) => {
    const current = newKeyData.allowedModels || [];
    if (current.includes(modelId)) {
      setNewKeyData({
        ...newKeyData,
        allowedModels: current.filter((m) => m !== modelId),
      });
    } else {
      setNewKeyData({
        ...newKeyData,
        allowedModels: [...current, modelId],
      });
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <Loader2 className="w-10 h-10 text-accent animate-spin" />
      </div>
    );
  }

  const hasSelection = selectedKeys.size > 0;
  const allSelected = apiKeys && apiKeys.length > 0 && selectedKeys.size === apiKeys.length;

  return (
    <div className="min-h-screen bg-bg-primary grain">
      <DashboardHeader />

      <main className="container-custom py-8">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="font-display text-heading-1 mb-2">API Keys</h1>
            <p className="text-text-secondary">管理您的 API 密钥</p>
          </div>
          <div className="flex items-center gap-3">
            {/* 批量操作按钮 */}
            {hasSelection && (
              <div className="flex items-center gap-2 animate-fade-in">
                <span className="text-sm text-text-tertiary">
                  已选择 {selectedKeys.size} 个密钥
                </span>
                <button
                  onClick={() => batchUpdateMutation.mutate({
                    ids: Array.from(selectedKeys),
                    status: 'active',
                  })}
                  disabled={batchUpdateMutation.isPending}
                  className="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-success/10 text-success text-sm font-medium hover:bg-success/20 transition-colors disabled:opacity-50"
                >
                  {batchUpdateMutation.isPending ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <CheckCircle2 className="w-4 h-4" />
                  )}
                  批量启用
                </button>
                <button
                  onClick={() => batchUpdateMutation.mutate({
                    ids: Array.from(selectedKeys),
                    status: 'disabled',
                  })}
                  disabled={batchUpdateMutation.isPending}
                  className="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-[#f59e0b]/10 text-[#f59e0b] text-sm font-medium hover:bg-[#f59e0b]/20 transition-colors disabled:opacity-50"
                >
                  {batchUpdateMutation.isPending ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Ban className="w-4 h-4" />
                  )}
                  批量禁用
                </button>
              </div>
            )}
            <button
              onClick={() => setIsCreateModalOpen(true)}
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all duration-300 hover:shadow-glow"
            >
              <Plus className="w-5 h-5" />
              创建新密钥
            </button>
          </div>
        </div>

        {/* API Keys 列表 */}
        <div className="bg-bg-tertiary rounded-2xl border border-white/5 overflow-hidden">
          {apiKeys && apiKeys.length > 0 ? (
            <>
              {/* 列表头 */}
              <div className="px-6 py-3 border-b border-white/5 flex items-center gap-4 bg-bg-secondary/30">
                <button
                  onClick={toggleSelectAll}
                  className="flex-shrink-0 w-5 h-5 rounded border border-white/20 flex items-center justify-center transition-colors hover:border-accent/50"
                >
                  {allSelected && (
                    <Check className="w-3.5 h-3.5 text-accent" />
                  )}
                </button>
                <span className="text-xs text-text-muted font-medium uppercase tracking-wider flex-1">密钥信息</span>
                <span className="text-xs text-text-muted font-medium uppercase tracking-wider w-48 text-right">使用统计</span>
                <span className="text-xs text-text-muted font-medium uppercase tracking-wider w-20 text-center">状态</span>
                <span className="text-xs text-text-muted font-medium uppercase tracking-wider w-24 text-right">操作</span>
              </div>

              <div className="divide-y divide-white/5">
                {apiKeys.map((key) => {
                  const stats = keyUsageMap?.[key.id];
                  const health = getHealthStatus(key, stats);
                  const healthConfig = HEALTH_CONFIG[health];
                  const HealthIcon = healthConfig.icon;
                  const isSelected = selectedKeys.has(key.id);

                  return (
                    <div
                      key={key.id}
                      className={`p-6 hover:bg-white/[0.02] transition-colors ${isSelected ? 'bg-accent/5' : ''}`}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex items-start gap-4 flex-1 min-w-0">
                          {/* 选择框 */}
                          <button
                            onClick={() => toggleKeySelection(key.id)}
                            className="flex-shrink-0 w-5 h-5 mt-1 rounded border border-white/20 flex items-center justify-center transition-colors hover:border-accent/50"
                          >
                            {isSelected && (
                              <Check className="w-3.5 h-3.5 text-accent" />
                            )}
                          </button>

                          <div className="p-3 rounded-xl bg-accent/10 flex-shrink-0">
                            <Key className="w-6 h-6 text-accent" />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-3 mb-1">
                              <h3 className="font-semibold text-lg truncate">{key.name}</h3>
                              {/* 健康状态指示器 */}
                              <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${healthConfig.bgColor} ${healthConfig.color}`}>
                                <span className={`w-1.5 h-1.5 rounded-full ${healthConfig.dotColor} ${health === 'warning' ? 'animate-pulse' : ''}`} />
                                {healthConfig.label}
                              </span>
                            </div>
                            <div className="flex items-center gap-2 text-text-tertiary text-sm mb-2">
                              <code className="px-2 py-1 bg-bg-secondary rounded text-text-secondary font-mono">
                                {key.keyPrefix}...
                              </code>
                              <button
                                onClick={() => handleCopyKey(key.keyPrefix, key.id)}
                                className="p-1 hover:bg-white/5 rounded transition-colors"
                                title="复制密钥前缀"
                              >
                                {copiedKeyId === key.id ? (
                                  <Check className="w-4 h-4 text-success" />
                                ) : (
                                  <Copy className="w-4 h-4" />
                                )}
                              </button>
                            </div>
                            <div className="flex items-center gap-4 text-sm text-text-tertiary">
                              <div className="flex items-center gap-1">
                                <Clock className="w-4 h-4" />
                                创建于 {formatDate(key.createdAt)}
                              </div>
                              {key.lastUsedAt && (
                                <div className="flex items-center gap-1">
                                  最后使用 {formatDate(key.lastUsedAt)}
                                </div>
                              )}
                            </div>
                            {(key.dailyLimit || key.monthlyLimit) && (
                              <div className="flex items-center gap-3 mt-2 text-sm">
                                {key.dailyLimit && (
                                  <span className="px-2 py-1 bg-accent-2/10 text-accent-2 rounded">
                                    日限额: {key.dailyLimit.toLocaleString()}
                                  </span>
                                )}
                                {key.monthlyLimit && (
                                  <span className="px-2 py-1 bg-accent/10 text-accent rounded">
                                    月限额: {key.monthlyLimit.toLocaleString()}
                                  </span>
                                )}
                              </div>
                            )}
                          </div>
                        </div>

                        {/* 使用统计 */}
                        <div className="w-48 flex-shrink-0 text-right">
                          {stats ? (
                            <div className="space-y-1.5">
                              <div className="flex items-center justify-end gap-1.5 text-sm">
                                <Activity className="w-3.5 h-3.5 text-text-muted" />
                                <span className="text-text-secondary font-medium">{formatNumber(stats.monthRequests)}</span>
                                <span className="text-text-muted text-xs">次/月</span>
                              </div>
                              <div className="flex items-center justify-end gap-1.5 text-sm">
                                <Hash className="w-3.5 h-3.5 text-text-muted" />
                                <span className="text-text-secondary font-medium">{formatNumber(stats.monthTokens)}</span>
                                <span className="text-text-muted text-xs">tokens</span>
                              </div>
                              <div className="flex items-center justify-end gap-1.5 text-sm">
                                <Coins className="w-3.5 h-3.5 text-text-muted" />
                                <span className="text-accent font-medium">{stats.monthCost.toFixed(2)}</span>
                                <span className="text-text-muted text-xs">CRED</span>
                              </div>
                            </div>
                          ) : (
                            <span className="text-xs text-text-muted">--</span>
                          )}
                        </div>

                        {/* 状态 */}
                        <div className="w-20 flex-shrink-0 flex justify-center">
                          <button
                            onClick={() => updateMutation.mutate({
                              id: key.id,
                              data: { status: key.status === 'active' ? 'disabled' : 'active' }
                            })}
                            className={`p-2 rounded-lg transition-colors ${
                              key.status === 'active'
                                ? 'text-success hover:bg-success/10'
                                : 'text-text-muted hover:bg-white/5'
                            }`}
                            title={key.status === 'active' ? '禁用' : '启用'}
                          >
                            {key.status === 'active' ? (
                              <ToggleRight className="w-6 h-6" />
                            ) : (
                              <ToggleLeft className="w-6 h-6" />
                            )}
                          </button>
                        </div>

                        {/* 操作 */}
                        <div className="w-24 flex-shrink-0 flex justify-end items-center gap-2">
                          {deleteConfirmId === key.id ? (
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => deleteMutation.mutate(key.id)}
                                className="px-3 py-1.5 rounded-lg bg-error text-white text-sm font-medium hover:bg-error/80 transition-colors"
                              >
                                确认
                              </button>
                              <button
                                onClick={() => setDeleteConfirmId(null)}
                                className="px-3 py-1.5 rounded-lg bg-white/5 text-text-secondary text-sm font-medium hover:bg-white/10 transition-colors"
                              >
                                取消
                              </button>
                            </div>
                          ) : (
                            <button
                              onClick={() => setDeleteConfirmId(key.id)}
                              className="p-2 rounded-lg text-text-tertiary hover:text-error hover:bg-error/10 transition-colors"
                              title="删除"
                            >
                              <Trash2 className="w-5 h-5" />
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </>
          ) : (
            <div className="p-12 text-center">
              <Key className="w-16 h-16 text-text-muted mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">暂无 API 密钥</h3>
              <p className="text-text-tertiary mb-6">创建您的第一个 API 密钥开始使用</p>
              <button
                onClick={() => setIsCreateModalOpen(true)}
                className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all duration-300"
              >
                <Plus className="w-5 h-5" />
                创建新密钥
              </button>
            </div>
          )}
        </div>
      </main>

      {/* 创建 API Key 模态框 */}
      {isCreateModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={handleCloseCreateModal} />
          <div className="relative bg-bg-tertiary rounded-2xl border border-white/10 p-6 w-full max-w-lg shadow-2xl max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h2 className="font-display text-xl font-bold">
                {createdKey ? '密钥已创建' : '创建新密钥'}
              </h2>
              <button
                onClick={handleCloseCreateModal}
                className="p-2 rounded-lg text-text-tertiary hover:text-text-primary hover:bg-white/5 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {createdKey ? (
              <div className="space-y-4">
                <div className="p-4 rounded-xl bg-success/10 border border-success/20">
                  <div className="flex items-start gap-3">
                    <Check className="w-5 h-5 text-success mt-0.5" />
                    <div>
                      <p className="font-medium text-success mb-1">密钥创建成功</p>
                      <p className="text-sm text-text-secondary">
                        请立即复制并安全保存您的密钥。此密钥只会显示一次。
                      </p>
                    </div>
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-bg-secondary border border-white/5">
                  <code className="block text-sm font-mono text-text-primary break-all select-all">
                    {createdKey}
                  </code>
                </div>

                <button
                  onClick={() => handleCopyKey(createdKey, 'new')}
                  className="w-full py-3 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all flex items-center justify-center gap-2"
                >
                  {copiedKeyId === 'new' ? (
                    <>
                      <Check className="w-5 h-5" />
                      已复制
                    </>
                  ) : (
                    <>
                      <Copy className="w-5 h-5" />
                      复制密钥
                    </>
                  )}
                </button>
              </div>
            ) : (
              <div className="space-y-5">
                {createMutation.error && (
                  <div className="p-4 rounded-xl bg-error/10 border border-error/20 text-error text-sm">
                    {(createMutation.error as any).response?.data?.message || '创建失败，请重试'}
                  </div>
                )}

                {/* 密钥名称 */}
                <div>
                  <label className="block text-sm font-medium mb-2 text-text-secondary">密钥名称</label>
                  <input
                    type="text"
                    value={newKeyData.name}
                    onChange={(e) => setNewKeyData({ ...newKeyData, name: e.target.value })}
                    className="input-field"
                    placeholder="例如：生产环境密钥"
                    disabled={createMutation.isPending}
                  />
                </div>

                {/* 允许的模型列表（多选） */}
                <div>
                  <label className="block text-sm font-medium mb-2 text-text-secondary">
                    <div className="flex items-center gap-2">
                      <Zap className="w-4 h-4 text-accent" />
                      允许的模型
                    </div>
                  </label>
                  <p className="text-xs text-text-muted mb-2">留空则允许所有模型</p>
                  <div className="flex flex-wrap gap-2">
                    {AVAILABLE_MODELS.map((model) => {
                      const isSelected = newKeyData.allowedModels?.includes(model.id);
                      return (
                        <button
                          key={model.id}
                          type="button"
                          onClick={() => toggleModelSelection(model.id)}
                          className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all border ${
                            isSelected
                              ? 'bg-accent/10 border-accent/30 text-accent'
                              : 'bg-bg-secondary border-white/5 text-text-secondary hover:border-white/20'
                          }`}
                        >
                          {model.name}
                          {isSelected && <Check className="w-3 h-3 ml-1 inline" />}
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* IP 白名单 */}
                <div>
                  <label className="block text-sm font-medium mb-2 text-text-secondary">
                    <div className="flex items-center gap-2">
                      <Globe className="w-4 h-4 text-accent-2" />
                      IP 白名单（可选）
                    </div>
                  </label>
                  <p className="text-xs text-text-muted mb-2">每行一个 IP 地址，留空则不限制</p>
                  <textarea
                    value={newKeyData.ipWhitelist || ''}
                    onChange={(e) => setNewKeyData({ ...newKeyData, ipWhitelist: e.target.value })}
                    className="input-field resize-none"
                    placeholder={'192.168.1.0/24\n10.0.0.1'}
                    rows={3}
                    disabled={createMutation.isPending}
                  />
                </div>

                {/* RPM 限制 */}
                <div>
                  <label className="block text-sm font-medium mb-2 text-text-secondary">
                    <div className="flex items-center gap-2">
                      <Shield className="w-4 h-4 text-[#f59e0b]" />
                      每分钟请求限制 (RPM)
                    </div>
                  </label>
                  <input
                    type="number"
                    value={newKeyData.rpmLimit || ''}
                    onChange={(e) => setNewKeyData({
                      ...newKeyData,
                      rpmLimit: e.target.value ? Number(e.target.value) : undefined,
                    })}
                    className="input-field"
                    placeholder="例如：60"
                    min="1"
                    disabled={createMutation.isPending}
                  />
                </div>

                {/* 每日限额 */}
                <div>
                  <label className="block text-sm font-medium mb-2 text-text-secondary">
                    每日限额 (可选)
                  </label>
                  <input
                    type="number"
                    value={newKeyData.dailyLimit || ''}
                    onChange={(e) => setNewKeyData({
                      ...newKeyData,
                      dailyLimit: e.target.value ? Number(e.target.value) : undefined,
                    })}
                    className="input-field"
                    placeholder="请求次数"
                    min="1"
                    disabled={createMutation.isPending}
                  />
                </div>

                {/* 每月限额 */}
                <div>
                  <label className="block text-sm font-medium mb-2 text-text-secondary">
                    每月限额 (可选)
                  </label>
                  <input
                    type="number"
                    value={newKeyData.monthlyLimit || ''}
                    onChange={(e) => setNewKeyData({
                      ...newKeyData,
                      monthlyLimit: e.target.value ? Number(e.target.value) : undefined,
                    })}
                    className="input-field"
                    placeholder="请求次数"
                    min="1"
                    disabled={createMutation.isPending}
                  />
                </div>

                <div className="flex gap-3 pt-2">
                  <button
                    onClick={handleCloseCreateModal}
                    className="flex-1 py-3 rounded-xl bg-white/5 text-text-primary font-medium hover:bg-white/10 transition-colors"
                    disabled={createMutation.isPending}
                  >
                    取消
                  </button>
                  <button
                    onClick={handleCreateKey}
                    disabled={!newKeyData.name.trim() || createMutation.isPending}
                    className="flex-1 py-3 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                  >
                    {createMutation.isPending ? (
                      <>
                        <Loader2 className="w-5 h-5 animate-spin" />
                        创建中...
                      </>
                    ) : (
                      '创建密钥'
                    )}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
