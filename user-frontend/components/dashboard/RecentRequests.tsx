'use client';

import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { UsageRecord } from '@/lib/api/client';

interface RecentRequestsProps {
  requests?: UsageRecord[];
}

export function RecentRequests({ requests = [] }: RecentRequestsProps) {
  const getProviderBadge = (provider: string) => {
    const styles: Record<string, string> = {
      'self_hosted_ds_v4': 'bg-accent/10 text-accent',
      'deepseek_api': 'bg-blue-500/10 text-blue-400',
      'gpt_4_turbo': 'bg-purple-500/10 text-purple-400',
      'claude_3_opus': 'bg-orange-500/10 text-orange-400',
      'openai': 'bg-green-500/10 text-green-400',
      'anthropic': 'bg-orange-500/10 text-orange-400',
    };
    return styles[provider] || 'bg-gray-500/10 text-gray-400';
  };

  const getRequestTypeBadge = (type: string) => {
    const styles: Record<string, string> = {
      'chat': 'bg-accent-2/10 text-accent-2',
      'embedding': 'bg-blue-500/10 text-blue-400',
      'image': 'bg-purple-500/10 text-purple-400',
    };
    return styles[type] || 'bg-gray-500/10 text-gray-400';
  };

  const formatLatency = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b border-white/5">
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">时间</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">模型</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">类型</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">Tokens</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">费用</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">供应商</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">延迟</th>
            <th className="text-left py-3 px-4 text-sm font-medium text-text-tertiary">状态</th>
          </tr>
        </thead>
        <tbody>
          {requests.length === 0 ? (
            <tr>
              <td colSpan={8} className="py-8 text-center text-text-muted">
                暂无请求记录
              </td>
            </tr>
          ) : (
            requests.map((req) => (
              <tr key={req.id} className="border-b border-white/5 hover:bg-white/[0.02] transition-colors">
                <td className="py-3 px-4 text-sm text-text-secondary">
                  {format(new Date(req.createdAt), 'MM-dd HH:mm', { locale: zhCN })}
                </td>
                <td className="py-3 px-4 text-sm font-medium">{req.model}</td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getRequestTypeBadge(req.requestType)}`}>
                    {req.requestType === 'chat' ? '对话' : req.requestType === 'embedding' ? '向量' : '图片'}
                  </span>
                </td>
                <td className="py-3 px-4 text-sm text-text-secondary">
                  <span className="text-text-tertiary">{req.inputTokens.toLocaleString()}</span>
                  {' / '}
                  <span className="text-accent-2">{req.outputTokens.toLocaleString()}</span>
                </td>
                <td className="py-3 px-4 text-sm font-medium text-accent">
                  {req.cost.toFixed(4)} CRED
                </td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getProviderBadge(req.provider)}`}>
                    {req.provider.replace(/_/g, ' ')}
                  </span>
                </td>
                <td className="py-3 px-4 text-sm text-text-secondary">
                  {formatLatency(req.latency)}
                </td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    req.status === 'success' 
                      ? 'bg-success/10 text-success' 
                      : 'bg-error/10 text-error'
                  }`}>
                    {req.status === 'success' ? '成功' : '失败'}
                  </span>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
