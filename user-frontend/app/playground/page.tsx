'use client';

import { useState, useRef, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Send, Loader2, Sparkles, Bot, User, ChevronDown,
  Key, Zap, Gauge, DollarSign, Tag
} from 'lucide-react';
import { apiClient, ApiKey } from '@/lib/api/client';
import { DashboardHeader } from '@/components/dashboard/DashboardHeader';

interface Message {
  role: 'user' | 'assistant';
  content: string;
}

interface RouterDecision {
  complexity_score: number;
  complexity_level?: string;
  recommended_model?: string;
  cost_saving_ratio?: number;
  target_provider: string;
  reason: string;
  confidence: number;
}

// 模型列表
const MODEL_OPTIONS = [
  { id: 'auto', name: 'Auto（智能路由）', tier: 'auto' },
  { id: 'deepseek-v4-pro', name: 'DeepSeek-V4-Pro', tier: 'standard' },
  { id: 'deepseek-v4-flash', name: 'DeepSeek-V4-Flash', tier: 'economy' },
  { id: 'claude-sonnet-4', name: 'Claude Sonnet 4', tier: 'standard' },
  { id: 'claude-opus-4', name: 'Claude Opus 4', tier: 'premium' },
  { id: 'gpt-4.1', name: 'GPT-4.1', tier: 'standard' },
  { id: 'gpt-4.1-mini', name: 'GPT-4.1 Mini', tier: 'economy' },
];

// 模型定价（每百万 token）
const MODEL_PRICING: Record<string, number> = {
  economy: 0.1,
  standard: 1.0,
  premium: 10.0,
};

// 复杂度级别配置
const COMPLEXITY_LEVELS: Record<string, { label: string; color: string; bgColor: string; range: [number, number] }> = {
  simple: { label: '简单', color: 'text-accent-2', bgColor: 'bg-accent-2/10', range: [0, 0.25] },
  medium: { label: '中等', color: 'text-[#3b82f6]', bgColor: 'bg-[#3b82f6]/10', range: [0.25, 0.5] },
  complex: { label: '复杂', color: 'text-accent', bgColor: 'bg-accent/10', range: [0.5, 0.75] },
  expert: { label: '专家', color: 'text-[#ef4444]', bgColor: 'bg-[#ef4444]/10', range: [0.75, 1.0] },
};

function getComplexityLevel(score: number): string {
  if (score < 0.25) return 'simple';
  if (score < 0.5) return 'medium';
  if (score < 0.75) return 'complex';
  return 'expert';
}

function getModelTier(modelId: string): string {
  const model = MODEL_OPTIONS.find((m) => m.id === modelId);
  return model?.tier || 'standard';
}

export default function PlaygroundPage() {
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [routerInfo, setRouterInfo] = useState<RouterDecision | null>(null);
  const [estimatedCost, setEstimatedCost] = useState(0);
  const [selectedModel, setSelectedModel] = useState('auto');
  const [isModelDropdownOpen, setIsModelDropdownOpen] = useState(false);
  const [selectedApiKeyId, setSelectedApiKeyId] = useState<string>('');
  const [isKeyDropdownOpen, setIsKeyDropdownOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const modelDropdownRef = useRef<HTMLDivElement>(null);
  const keyDropdownRef = useRef<HTMLDivElement>(null);

  // 获取 API Key 列表（优雅降级）
  const { data: apiKeys } = useQuery({
    queryKey: ['api-keys-list'],
    queryFn: () => apiClient.listApiKeys(),
    retry: 1,
    staleTime: 30000,
  });

  // 从 localStorage 恢复选中的 API Key
  useEffect(() => {
    const savedKey = localStorage.getItem('selected-api-key-id');
    if (savedKey) {
      setSelectedApiKeyId(savedKey);
    }
  }, []);

  // 点击外部关闭下拉菜单
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (modelDropdownRef.current && !modelDropdownRef.current.contains(event.target as Node)) {
        setIsModelDropdownOpen(false);
      }
      if (keyDropdownRef.current && !keyDropdownRef.current.contains(event.target as Node)) {
        setIsKeyDropdownOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSelectApiKey = (keyId: string, keyPrefix: string) => {
    setSelectedApiKeyId(keyId);
    localStorage.setItem('selected-api-key-id', keyId);
    localStorage.setItem('api-key', keyPrefix);
    setIsKeyDropdownOpen(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isStreaming) return;

    const userMessage: Message = { role: 'user', content: input };
    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setIsStreaming(true);
    setRouterInfo(null);
    setEstimatedCost(0);

    let assistantContent = '';
    // 确定使用的模型 tier
    const modelTier = getModelTier(selectedModel);

    try {
      for await (const chunk of apiClient.streamChat({
        model: selectedModel,
        messages: [...messages, userMessage].map((m) => ({
          role: m.role,
          content: m.content,
        })),
      })) {
        if (chunk.choices?.[0]?.delta?.content) {
          assistantContent += chunk.choices[0].delta.content;
          setMessages((prev) => {
            const newMessages = [...prev];
            const lastMessage = newMessages[newMessages.length - 1];
            if (lastMessage?.role === 'assistant') {
              lastMessage.content = assistantContent;
            } else {
              newMessages.push({ role: 'assistant', content: assistantContent });
            }
            return newMessages;
          });
        }

        if (chunk.router_decision) {
          const decision = chunk.router_decision;
          // 补充复杂度分析信息
          const level = decision.complexity_level || getComplexityLevel(decision.complexity_score);
          setRouterInfo({
            ...decision,
            complexity_level: level,
            recommended_model: decision.recommended_model || decision.target_provider,
            cost_saving_ratio: decision.cost_saving_ratio,
          });
        }

        if (chunk.usage) {
          // 动态费用计算：基于模型 tier 和 token 数
          const pricing = MODEL_PRICING[modelTier] || MODEL_PRICING.standard;
          const cost = (chunk.usage.total_tokens / 1000000) * pricing;
          setEstimatedCost(cost);
        }
      }
    } catch (error) {
      console.error('Stream error:', error);
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: '抱歉，请求处理失败，请重试。' },
      ]);
    } finally {
      setIsStreaming(false);
    }
  };

  const currentModelOption = MODEL_OPTIONS.find((m) => m.id === selectedModel);
  const selectedApiKey = apiKeys?.find((k) => k.id === selectedApiKeyId);

  return (
    <div className="min-h-screen bg-bg-primary grain flex flex-col">
      <DashboardHeader />

      {/* API Key 选择器栏 */}
      <div className="border-b border-white/5 bg-bg-secondary/30 px-6 py-3">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="relative" ref={keyDropdownRef}>
              <button
                onClick={() => setIsKeyDropdownOpen(!isKeyDropdownOpen)}
                className="flex items-center gap-2 px-4 py-2 rounded-xl bg-bg-tertiary border border-white/10 text-sm text-text-secondary hover:border-white/20 transition-colors"
              >
                <Key className="w-4 h-4 text-accent" />
                <span className="max-w-[200px] truncate">
                  {selectedApiKey
                    ? `${selectedApiKey.name} (${selectedApiKey.keyPrefix}...)`
                    : '选择 API Key'}
                </span>
                <ChevronDown className="w-4 h-4 text-text-muted" />
              </button>
              {isKeyDropdownOpen && (
                <div className="absolute top-full left-0 mt-2 w-72 bg-bg-tertiary border border-white/10 rounded-xl shadow-card overflow-hidden z-50">
                  <div className="p-2 border-b border-white/5">
                    <p className="text-xs text-text-muted px-2">选择 API Key 用于请求</p>
                  </div>
                  <div className="max-h-60 overflow-y-auto p-1">
                    {apiKeys && apiKeys.length > 0 ? (
                      apiKeys
                        .filter((k) => k.status === 'active')
                        .map((key) => (
                          <button
                            key={key.id}
                            onClick={() => handleSelectApiKey(key.id, key.keyPrefix)}
                            className={`w-full text-left px-3 py-2.5 rounded-lg text-sm transition-colors flex items-center gap-3 ${
                              selectedApiKeyId === key.id
                                ? 'bg-accent/10 text-accent'
                                : 'text-text-secondary hover:bg-white/5'
                            }`}
                          >
                            <Key className="w-4 h-4 flex-shrink-0" />
                            <div className="min-w-0">
                              <p className="font-medium truncate">{key.name}</p>
                              <p className="text-xs text-text-muted font-mono">{key.keyPrefix}...</p>
                            </div>
                          </button>
                        ))
                    ) : (
                      <div className="px-3 py-4 text-center text-text-muted text-sm">
                        暂无可用密钥
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
            {selectedApiKey && (
              <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-success/10 text-success">
                已连接
              </span>
            )}
          </div>

          {/* 当前模型信息 */}
          <div className="flex items-center gap-2 text-sm text-text-tertiary">
            <Zap className="w-4 h-4 text-accent" />
            <span>当前模型：</span>
            <span className="text-text-primary font-medium">{currentModelOption?.name || 'Auto'}</span>
          </div>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Main Chat Area */}
        <div className="flex-1 flex flex-col">
          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-6">
            {messages.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-text-tertiary">
                <div className="w-20 h-20 rounded-2xl bg-accent/10 flex items-center justify-center mb-6">
                  <Sparkles className="w-10 h-10 text-accent" />
                </div>
                <h3 className="font-display text-2xl font-semibold mb-3 text-text-primary">开始对话</h3>
                <p className="text-center max-w-md text-body">
                  输入您的问题，AI 将自动选择最优模型进行回答
                </p>
                <div className="mt-6 flex flex-wrap gap-2 justify-center">
                  {['写一首诗', '解释量子计算', '帮我写代码', '分析数据趋势'].map((suggestion) => (
                    <button
                      key={suggestion}
                      onClick={() => setInput(suggestion)}
                      className="px-4 py-2 rounded-xl bg-bg-tertiary border border-white/5 text-sm text-text-secondary hover:border-accent/30 hover:text-accent transition-colors"
                    >
                      {suggestion}
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="max-w-3xl mx-auto space-y-6">
                {messages.map((message, index) => (
                  <div
                    key={index}
                    className={`flex gap-4 ${
                      message.role === 'user' ? 'flex-row-reverse' : ''
                    }`}
                  >
                    <div className={`w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 ${
                      message.role === 'user'
                        ? 'bg-accent/10 text-accent'
                        : 'bg-accent-2/10 text-accent-2'
                    }`}>
                      {message.role === 'user' ? <User size={20} /> : <Bot size={20} />}
                    </div>
                    <div
                      className={`max-w-[80%] rounded-2xl px-5 py-4 ${
                        message.role === 'user'
                          ? 'bg-accent text-white'
                          : 'bg-bg-tertiary border border-white/5'
                      }`}
                    >
                      <p className="whitespace-pre-wrap text-body leading-relaxed">{message.content}</p>
                    </div>
                  </div>
                ))}
                {isStreaming && (
                  <div className="flex gap-4">
                    <div className="w-10 h-10 rounded-xl bg-accent-2/10 flex items-center justify-center">
                      <Bot size={20} className="text-accent-2" />
                    </div>
                    <div className="bg-bg-tertiary border border-white/5 rounded-2xl px-5 py-4">
                      <div className="flex gap-1.5">
                        <span className="w-2 h-2 bg-accent rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                        <span className="w-2 h-2 bg-accent rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                        <span className="w-2 h-2 bg-accent rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                      </div>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>
            )}
          </div>

          {/* Input Area */}
          <div className="border-t border-white/5 p-6 bg-bg-secondary/50">
            <div className="max-w-3xl mx-auto">
              {/* 模型选择器 */}
              <div className="mb-3" ref={modelDropdownRef}>
                <div className="relative">
                  <button
                    onClick={() => setIsModelDropdownOpen(!isModelDropdownOpen)}
                    className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-bg-tertiary border border-white/10 text-sm text-text-secondary hover:border-white/20 transition-colors"
                  >
                    <Zap className="w-3.5 h-3.5 text-accent" />
                    <span>{currentModelOption?.name || 'Auto（智能路由）'}</span>
                    <ChevronDown className={`w-3.5 h-3.5 text-text-muted transition-transform ${isModelDropdownOpen ? 'rotate-180' : ''}`} />
                  </button>
                  {isModelDropdownOpen && (
                    <div className="absolute bottom-full left-0 mb-2 w-64 bg-bg-tertiary border border-white/10 rounded-xl shadow-card overflow-hidden z-50">
                      <div className="p-2 border-b border-white/5">
                        <p className="text-xs text-text-muted px-2">选择模型</p>
                      </div>
                      <div className="p-1">
                        {MODEL_OPTIONS.map((model) => (
                          <button
                            key={model.id}
                            onClick={() => {
                              setSelectedModel(model.id);
                              setIsModelDropdownOpen(false);
                            }}
                            className={`w-full text-left px-3 py-2.5 rounded-lg text-sm transition-colors flex items-center justify-between ${
                              selectedModel === model.id
                                ? 'bg-accent/10 text-accent'
                                : 'text-text-secondary hover:bg-white/5'
                            }`}
                          >
                            <div className="flex items-center gap-2">
                              <Zap className={`w-4 h-4 ${
                                model.tier === 'economy' ? 'text-accent-2' :
                                model.tier === 'standard' ? 'text-accent' :
                                model.tier === 'premium' ? 'text-[#f59e0b]' :
                                'text-text-muted'
                              }`} />
                              <span>{model.name}</span>
                            </div>
                            {model.tier !== 'auto' && (
                              <span className={`text-xs px-1.5 py-0.5 rounded ${
                                model.tier === 'economy' ? 'bg-accent-2/10 text-accent-2' :
                                model.tier === 'standard' ? 'bg-accent/10 text-accent' :
                                'bg-[#f59e0b]/10 text-[#f59e0b]'
                              }`}>
                                ${MODEL_PRICING[model.tier]}/M
                              </span>
                            )}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>

              <form onSubmit={handleSubmit} className="relative">
                <textarea
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      handleSubmit(e);
                    }
                  }}
                  placeholder="输入您的问题..."
                  className="w-full bg-bg-tertiary border border-white/10 rounded-2xl pl-5 pr-14 py-4 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20 resize-none text-body text-text-primary placeholder:text-text-muted"
                  rows={3}
                  disabled={isStreaming}
                />
                <button
                  type="submit"
                  disabled={!input.trim() || isStreaming}
                  className="absolute right-3 bottom-3 p-2.5 rounded-xl bg-accent hover:bg-accent-hover text-white disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-300 hover:shadow-glow"
                >
                  {isStreaming ? (
                    <Loader2 className="w-5 h-5 animate-spin" />
                  ) : (
                    <Send className="w-5 h-5" />
                  )}
                </button>
              </form>

              {estimatedCost > 0 && (
                <div className="mt-3 flex items-center justify-between text-xs text-text-muted">
                  <span>
                    模型：<span className="text-text-secondary font-medium">{currentModelOption?.name}</span>
                    {' '}&middot;{' '}
                    定价：<span className="text-accent font-medium">${MODEL_PRICING[getModelTier(selectedModel)]}/M tokens</span>
                  </span>
                  <span>
                    预估费用: <span className="text-accent font-medium">{estimatedCost.toFixed(6)} CRED</span>
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Router Info Sidebar */}
        {routerInfo && (
          <div className="w-80 border-l border-white/5 p-6 hidden lg:block bg-bg-secondary/30 overflow-y-auto">
            <h3 className="font-display text-heading-3 mb-6">路由决策</h3>

            <div className="space-y-5">
              {/* 复杂度分析面板 */}
              <div className="bg-bg-tertiary rounded-xl p-5 border border-white/5">
                <div className="flex items-center gap-2 mb-4">
                  <Gauge className="w-4 h-4 text-accent" />
                  <p className="text-body-small font-medium text-text-secondary">复杂度分析</p>
                </div>

                {/* 复杂度评分标尺 */}
                <div className="mb-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs text-text-muted">复杂度评分</span>
                    <span className="font-display text-lg font-bold text-text-primary">
                      {routerInfo.complexity_score.toFixed(2)}
                    </span>
                  </div>
                  <div className="relative h-3 bg-bg-elevated rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all duration-500"
                      style={{
                        width: `${routerInfo.complexity_score * 100}%`,
                        background: routerInfo.complexity_score < 0.25
                          ? '#14b8a6'
                          : routerInfo.complexity_score < 0.5
                          ? '#3b82f6'
                          : routerInfo.complexity_score < 0.75
                          ? '#ff6b35'
                          : '#ef4444',
                      }}
                    />
                    {/* 刻度标记 */}
                    <div className="absolute inset-0 flex">
                      <div className="flex-1 border-r border-white/10" />
                      <div className="flex-1 border-r border-white/10" />
                      <div className="flex-1 border-r border-white/10" />
                      <div className="flex-1" />
                    </div>
                  </div>
                  <div className="flex justify-between mt-1 text-[10px] text-text-muted">
                    <span>0</span>
                    <span>0.25</span>
                    <span>0.5</span>
                    <span>0.75</span>
                    <span>1.0</span>
                  </div>
                </div>

                {/* 复杂度级别标签 */}
                {routerInfo.complexity_level && (
                  <div className="mb-4">
                    <span className="text-xs text-text-muted mb-1.5 block">复杂度级别</span>
                    <div className="flex gap-2">
                      {Object.entries(COMPLEXITY_LEVELS).map(([key, config]) => (
                        <span
                          key={key}
                          className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-all ${
                            routerInfo.complexity_level === key
                              ? `${config.bgColor} ${config.color} ring-1 ring-current/20`
                              : 'bg-bg-elevated text-text-muted'
                          }`}
                        >
                          {config.label}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {/* 推荐模型 */}
                {routerInfo.recommended_model && (
                  <div className="mb-4">
                    <div className="flex items-center gap-2 mb-1.5">
                      <Zap className="w-3.5 h-3.5 text-accent" />
                      <span className="text-xs text-text-muted">推荐模型</span>
                    </div>
                    <p className="text-sm font-medium text-text-primary">
                      {routerInfo.recommended_model.replace(/_/g, ' ')}
                    </p>
                  </div>
                )}

                {/* 成本节省比例 */}
                {routerInfo.cost_saving_ratio !== undefined && (
                  <div>
                    <div className="flex items-center gap-2 mb-1.5">
                      <DollarSign className="w-3.5 h-3.5 text-accent-2" />
                      <span className="text-xs text-text-muted">成本节省</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 h-2 bg-bg-elevated rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-accent-2 to-accent rounded-full transition-all duration-500"
                          style={{ width: `${Math.min(routerInfo.cost_saving_ratio * 100, 100)}%` }}
                        />
                      </div>
                      <span className="text-accent-2 font-display font-bold text-sm">
                        {(routerInfo.cost_saving_ratio * 100).toFixed(0)}%
                      </span>
                    </div>
                  </div>
                )}
              </div>

              {/* 目标供应商 */}
              <div className="bg-bg-tertiary rounded-xl p-5 border border-white/5">
                <p className="text-body-small text-text-tertiary mb-2">目标供应商</p>
                <p className="font-medium text-text-primary">{routerInfo.target_provider.replace(/_/g, ' ')}</p>
              </div>

              {/* 置信度 */}
              <div className="bg-bg-tertiary rounded-xl p-5 border border-white/5">
                <p className="text-body-small text-text-tertiary mb-2">置信度</p>
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-2 bg-bg-elevated rounded-full overflow-hidden">
                    <div
                      className="h-full bg-accent-2"
                      style={{ width: `${routerInfo.confidence * 100}%` }}
                    />
                  </div>
                  <span className="text-accent-2 font-medium">{(routerInfo.confidence * 100).toFixed(0)}%</span>
                </div>
              </div>

              {/* 路由原因 */}
              <div className="bg-bg-tertiary rounded-xl p-5 border border-white/5">
                <p className="text-body-small text-text-tertiary mb-2">路由原因</p>
                <p className="text-body-small text-text-secondary leading-relaxed">{routerInfo.reason}</p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
