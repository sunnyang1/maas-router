'use client';

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { CostCard } from '@/components/dashboard/CostCard';
import { RequestChart } from '@/components/dashboard/RequestChart';
import { RouterPieChart } from '@/components/dashboard/RouterPieChart';
import { RecentRequests } from '@/components/dashboard/RecentRequests';
import { DashboardHeader } from '@/components/dashboard/DashboardHeader';
import { Loader2, Zap, Target, TrendingDown, Activity } from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  Cell,
} from 'recharts';

interface ComplexityStats {
  monthlyRoutingCount: number;
  avgCostSaving: number;
  routingAccuracy: number;
}

const MODEL_BAR_COLORS = ['#14b8a6', '#ff6b35', '#f59e0b'];

export default function DashboardPage() {
  const { data: dashboardData, isLoading, error } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => apiClient.getDashboard(),
    refetchInterval: 30000,
  });

  // 获取复杂度分析统计（优雅降级）
  const { data: complexityStats } = useQuery({
    queryKey: ['complexity-stats'],
    queryFn: async (): Promise<ComplexityStats> => {
      const response = await apiClient.client.get('/complexity/stats');
      return response.data;
    },
    refetchInterval: 60000,
    retry: 1,
    staleTime: 60000,
  });

  // 模型使用分布数据（从 dashboard 数据中提取或使用默认值）
  const modelUsageData = dashboardData?.modelUsage || [
    { name: 'Economy', requests: 1240, color: '#14b8a6' },
    { name: 'Standard', requests: 856, color: '#ff6b35' },
    { name: 'Premium', requests: 312, color: '#f59e0b' },
  ];

  if (isLoading) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <Loader2 className="w-10 h-10 text-accent animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <div className="text-center">
          <p className="text-error mb-4">加载失败，请刷新页面重试</p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors"
          >
            刷新页面
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-bg-primary grain">
      <DashboardHeader />

      <main className="container-custom py-8">
        <div className="mb-8">
          <h1 className="font-display text-heading-1 mb-2">仪表盘</h1>
          <p className="text-text-secondary">实时监控您的 API 使用情况和成本</p>
        </div>

        <div className="grid grid-cols-12 gap-6">
          {/* Cost Cards */}
          <div className="col-span-12 md:col-span-4">
            <CostCard
              title="今日消费"
              amount={dashboardData?.todayCost || 0}
              currency="CRED"
              trend={dashboardData?.costTrend}
            />
          </div>
          <div className="col-span-12 md:col-span-4">
            <CostCard
              title="本周消费"
              amount={dashboardData?.weekCost || 0}
              currency="CRED"
              trend={dashboardData?.weekTrend}
            />
          </div>
          <div className="col-span-12 md:col-span-4">
            <CostCard
              title="本月消费"
              amount={dashboardData?.monthCost || 0}
              currency="CRED"
              trend={dashboardData?.monthTrend}
            />
          </div>

          {/* 智能路由概览卡片 */}
          {complexityStats && (
            <div className="col-span-12">
              <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="p-2 rounded-xl bg-accent/10">
                    <Zap className="w-5 h-5 text-accent" />
                  </div>
                  <h3 className="font-display text-heading-3">智能路由概览</h3>
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-accent-2/10 text-accent-2">
                    本月
                  </span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  {/* 本月智能路由次数 */}
                  <div className="bg-bg-secondary rounded-xl p-5 border border-white/5">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="p-2 rounded-lg bg-accent-2/10">
                        <Activity className="w-4 h-4 text-accent-2" />
                      </div>
                      <p className="text-body-small text-text-tertiary">智能路由次数</p>
                    </div>
                    <p className="font-display text-2xl font-bold text-text-primary">
                      {complexityStats.monthlyRoutingCount.toLocaleString()}
                    </p>
                    <p className="text-xs text-text-muted mt-1">次请求通过智能路由处理</p>
                  </div>

                  {/* 平均成本节省比例 */}
                  <div className="bg-bg-secondary rounded-xl p-5 border border-white/5">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="p-2 rounded-lg bg-accent/10">
                        <TrendingDown className="w-4 h-4 text-accent" />
                      </div>
                      <p className="text-body-small text-text-tertiary">平均成本节省</p>
                    </div>
                    <div className="flex items-baseline gap-1">
                      <span className="font-display text-2xl font-bold text-accent">
                        {complexityStats.avgCostSaving.toFixed(1)}%
                      </span>
                    </div>
                    <p className="text-xs text-text-muted mt-1">相比直接使用 Premium 模型</p>
                  </div>

                  {/* 路由准确率 */}
                  <div className="bg-bg-secondary rounded-xl p-5 border border-white/5">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="p-2 rounded-lg bg-accent-2/10">
                        <Target className="w-4 h-4 text-accent-2" />
                      </div>
                      <p className="text-body-small text-text-tertiary">路由准确率</p>
                    </div>
                    <div className="flex items-baseline gap-1">
                      <span className="font-display text-2xl font-bold text-accent-2">
                        {complexityStats.routingAccuracy.toFixed(1)}%
                      </span>
                    </div>
                    <p className="text-xs text-text-muted mt-1">模型选择与用户满意度匹配</p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Request Chart */}
          <div className="col-span-12 lg:col-span-8">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6">
              <h3 className="font-display text-heading-3 mb-6">请求量趋势</h3>
              <RequestChart data={dashboardData?.requestHistory} />
            </div>
          </div>

          {/* 模型使用分布柱状图 */}
          <div className="col-span-12 lg:col-span-4">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6">
              <h3 className="font-display text-heading-3 mb-6">模型使用分布</h3>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={modelUsageData} layout="vertical" margin={{ left: 0, right: 20 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#334155" horizontal={false} />
                    <XAxis
                      type="number"
                      stroke="#64748b"
                      fontSize={12}
                      tickLine={false}
                      axisLine={false}
                    />
                    <YAxis
                      type="category"
                      dataKey="name"
                      stroke="#64748b"
                      fontSize={12}
                      tickLine={false}
                      axisLine={false}
                      width={70}
                    />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#1e293b',
                        border: '1px solid #334155',
                        borderRadius: '8px',
                      }}
                      labelStyle={{ color: '#94a3b8' }}
                      formatter={(value: number) => [`${value.toLocaleString()} 次`, '请求数']}
                    />
                    <Bar dataKey="requests" name="请求数" radius={[0, 6, 6, 0]} barSize={24}>
                      {modelUsageData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={MODEL_BAR_COLORS[index % MODEL_BAR_COLORS.length]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Router Distribution */}
          <div className="col-span-12 lg:col-span-6">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6">
              <h3 className="font-display text-heading-3 mb-6">路由分布</h3>
              <RouterPieChart data={dashboardData?.routerDistribution} />
            </div>
          </div>

          {/* Recent Requests */}
          <div className="col-span-12 lg:col-span-6">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6">
              <h3 className="font-display text-heading-3 mb-6">最近请求</h3>
              <RecentRequests requests={dashboardData?.recentRequests} />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
