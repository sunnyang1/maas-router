'use client';

import { useState, useEffect } from 'react';
import {
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Loader2, TrendingDown, Zap, Target, CheckCircle } from 'lucide-react';

// ============== 类型定义 ==============

interface ComplexityStats {
  totalRequests: number;
  averageComplexity: number;
  averageCostSaving: number;
  qualityPassRate: number;
  complexityDistribution: Array<{
    level: string;
    count: number;
    percentage: number;
  }>;
  costSavingTrend: Array<{
    date: string;
    saving: number;
    requests: number;
  }>;
  modelUsageDistribution: Array<{
    tier: string;
    count: number;
    percentage: number;
  }>;
}

// ============== 常量 ==============

const COMPLEXITY_COLORS: Record<string, string> = {
  simple: '#22c55e',
  medium: '#14b8a6',
  complex: '#ff6b35',
  expert: '#ef4444',
};

const COMPLEXITY_LABELS: Record<string, string> = {
  simple: '简单',
  medium: '中等',
  complex: '复杂',
  expert: '专家',
};

const MODEL_TIER_COLORS: Record<string, string> = {
  economy: '#14b8a6',
  standard: '#ff6b35',
  premium: '#a855f7',
};

const MODEL_TIER_LABELS: Record<string, string> = {
  economy: '经济型',
  standard: '标准型',
  premium: '高端型',
};

// ============== 自定义 Tooltip ==============

const CustomTooltip = ({ active, payload, label }: any) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-bg-elevated border border-white/10 rounded-lg px-4 py-3 shadow-card">
        <p className="text-text-secondary text-caption mb-1">{label}</p>
        {payload.map((entry: any, index: number) => (
          <p key={index} className="text-body-small" style={{ color: entry.color }}>
            {entry.name}: {typeof entry.value === 'number' ? entry.value.toFixed(2) : entry.value}
            {entry.name === '成本节省' && '%'}
          </p>
        ))}
      </div>
    );
  }
  return null;
};

// ============== 主组件 ==============

export default function ComplexityDashboardPage() {
  const [stats, setStats] = useState<ComplexityStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await fetch('/api/v1/complexity/stats?period=7d');
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        const data = await response.json();
        setStats(data);
      } catch (err) {
        console.error('Failed to fetch complexity stats:', err);
        setError(err instanceof Error ? err.message : '加载失败');
      } finally {
        setLoading(false);
      }
    };

    fetchStats();

    // 每 30 秒自动刷新
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, []);

  // 加载状态
  if (loading) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <Loader2 className="w-10 h-10 text-accent animate-spin" />
      </div>
    );
  }

  // 错误状态
  if (error) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <div className="text-center">
          <p className="text-error mb-4">加载失败，请刷新页面重试</p>
          <p className="text-text-tertiary text-body-small mb-6">{error}</p>
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

  // 构造图表数据
  const pieData = (stats?.complexityDistribution || []).map((item) => ({
    name: COMPLEXITY_LABELS[item.level] || item.level,
    value: item.count,
    color: COMPLEXITY_COLORS[item.level] || '#666',
  }));

  const trendData = (stats?.costSavingTrend || []).map((item) => ({
    date: item.date,
    成本节省: item.saving,
    请求数: item.requests,
  }));

  const barData = (stats?.modelUsageDistribution || []).map((item) => ({
    name: MODEL_TIER_LABELS[item.tier] || item.tier,
    请求数: item.count,
    color: MODEL_TIER_COLORS[item.tier] || '#666',
  }));

  return (
    <div className="min-h-screen bg-bg-primary grain">
      {/* 页面标题 */}
      <div className="container-custom py-8">
        <div className="mb-8">
          <h1 className="font-display text-heading-1 mb-2">成本优化分析</h1>
          <p className="text-text-secondary">
            基于智能复杂度评估的模型路由优化，降低 API 调用成本
          </p>
        </div>

        {/* 关键指标卡片 */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <MetricCard
            icon={<Zap className="w-5 h-5" />}
            iconColor="text-accent"
            iconBg="bg-accent-muted"
            title="总请求数"
            value={stats?.totalRequests?.toLocaleString() || '0'}
            subtitle="最近 7 天"
          />
          <MetricCard
            icon={<Target className="w-5 h-5" />}
            iconColor="text-accent-2"
            iconBg="bg-accent-2-muted"
            title="平均复杂度"
            value={stats?.averageComplexity?.toFixed(2) || '0.00'}
            subtitle="0-1 评分范围"
          />
          <MetricCard
            icon={<TrendingDown className="w-5 h-5" />}
            iconColor="text-success"
            iconBg="bg-green-500/20"
            title="平均成本节省"
            value={`${stats?.averageCostSaving?.toFixed(1) || '0'}%`}
            subtitle="对比高端模型"
          />
          <MetricCard
            icon={<CheckCircle className="w-5 h-5" />}
            iconColor="text-warning"
            iconBg="bg-yellow-500/20"
            title="质量通过率"
            value={`${stats?.qualityPassRate?.toFixed(1) || '0'}%`}
            subtitle="自动质检通过"
          />
        </div>

        {/* 图表区域 */}
        <div className="grid grid-cols-12 gap-6">
          {/* 复杂度分布饼图 */}
          <div className="col-span-12 lg:col-span-4">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6 card-hover">
              <h3 className="font-display text-heading-3 mb-6">复杂度分布</h3>
              <div className="h-[280px]">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={pieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={60}
                      outerRadius={100}
                      paddingAngle={4}
                      dataKey="value"
                      stroke="none"
                    >
                      {pieData.map((entry, index) => (
                        <Cell key={index} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip content={<CustomTooltip />} />
                    <Legend
                      verticalAlign="bottom"
                      iconType="circle"
                      iconSize={8}
                      formatter={(value: string) => (
                        <span className="text-text-secondary text-body-small">{value}</span>
                      )}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* 成本节省趋势折线图 */}
          <div className="col-span-12 lg:col-span-8">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6 card-hover">
              <h3 className="font-display text-heading-3 mb-6">成本节省趋势</h3>
              <div className="h-[280px]">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={trendData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                    <XAxis
                      dataKey="date"
                      tick={{ fill: '#737373', fontSize: 12 }}
                      axisLine={{ stroke: 'rgba(255,255,255,0.1)' }}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fill: '#737373', fontSize: 12 }}
                      axisLine={{ stroke: 'rgba(255,255,255,0.1)' }}
                      tickLine={false}
                    />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend
                      verticalAlign="top"
                      align="right"
                      iconType="line"
                      iconSize={8}
                      formatter={(value: string) => (
                        <span className="text-text-secondary text-body-small">{value}</span>
                      )}
                    />
                    <Line
                      type="monotone"
                      dataKey="成本节省"
                      stroke="#ff6b35"
                      strokeWidth={2}
                      dot={{ r: 4, fill: '#ff6b35', strokeWidth: 0 }}
                      activeDot={{ r: 6, fill: '#ff6b35', stroke: '#0c0c0c', strokeWidth: 2 }}
                    />
                    <Line
                      type="monotone"
                      dataKey="请求数"
                      stroke="#14b8a6"
                      strokeWidth={2}
                      dot={{ r: 4, fill: '#14b8a6', strokeWidth: 0 }}
                      activeDot={{ r: 6, fill: '#14b8a6', stroke: '#0c0c0c', strokeWidth: 2 }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* 模型使用分布柱状图 */}
          <div className="col-span-12">
            <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6 card-hover">
              <h3 className="font-display text-heading-3 mb-6">模型使用分布</h3>
              <div className="h-[280px]">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={barData} barCategoryGap="20%">
                    <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                    <XAxis
                      dataKey="name"
                      tick={{ fill: '#737373', fontSize: 12 }}
                      axisLine={{ stroke: 'rgba(255,255,255,0.1)' }}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fill: '#737373', fontSize: 12 }}
                      axisLine={{ stroke: 'rgba(255,255,255,0.1)' }}
                      tickLine={false}
                    />
                    <Tooltip content={<CustomTooltip />} />
                    <Bar dataKey="请求数" radius={[8, 8, 0, 0]}>
                      {barData.map((entry, index) => (
                        <Cell key={index} fill={entry.color} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============== 指标卡片子组件 ==============

function MetricCard({
  icon,
  iconColor,
  iconBg,
  title,
  value,
  subtitle,
}: {
  icon: React.ReactNode;
  iconColor: string;
  iconBg: string;
  title: string;
  value: string;
  subtitle: string;
}) {
  return (
    <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-5 card-hover">
      <div className="flex items-start justify-between mb-3">
        <div className={`${iconBg} p-2.5 rounded-xl`}>
          <div className={iconColor}>{icon}</div>
        </div>
      </div>
      <p className="text-text-secondary text-body-small mb-1">{title}</p>
      <p className="font-display text-heading-2 mb-1">{value}</p>
      <p className="text-text-muted text-caption">{subtitle}</p>
    </div>
  );
}
