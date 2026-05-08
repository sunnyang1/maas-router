import { useEffect, useState } from "react";
import {
  Users,
  Activity,
  DollarSign,
  Key,
  TrendingUp,
  BarChart3,
} from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
} from "recharts";
import { dashboardApi } from "../../services/api";

const COLORS = ["#a855f7", "#3b82f6", "#22c55e", "#eab308", "#ef4444", "#ec4899", "#06b6d4", "#f97316"];

export default function DashboardPage() {
  const [overview, setOverview] = useState<any>(null);
  const [trends, setTrends] = useState<any[]>([]);
  const [modelDist, setModelDist] = useState<any[]>([]);
  const [recentRequests, setRecentRequests] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [ov, tr, md, rr] = await Promise.all([
          dashboardApi.overview(),
          dashboardApi.trends(),
          dashboardApi.modelDistribution(),
          dashboardApi.recentRequests(),
        ]);
        setOverview(ov);
        setTrends(tr.data || []);
        setModelDist(md.data || []);
        setRecentRequests(rr.data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  const stats = [
    { label: "总用户数", value: overview?.total_users || 0, icon: Users, color: "text-purple-400", bg: "bg-purple-500/10" },
    { label: "日活用户", value: overview?.active_today || 0, icon: Activity, color: "text-blue-400", bg: "bg-blue-500/10" },
    { label: "今日收入", value: `$${(overview?.today_revenue || 0).toLocaleString()}`, icon: DollarSign, color: "text-green-400", bg: "bg-green-500/10" },
    { label: "活跃 API Key", value: overview?.active_api_keys || 0, icon: Key, color: "text-yellow-400", bg: "bg-yellow-500/10" },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-white mb-8">管理概览</h1>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {stats.map((stat) => (
          <div
            key={stat.label}
            className="bg-[#111118] border border-[#27272a] rounded-xl p-6"
          >
            <div className="flex items-center justify-between mb-4">
              <div className={`p-2 rounded-lg ${stat.bg}`}>
                <stat.icon className={`w-5 h-5 ${stat.color}`} />
              </div>
            </div>
            <p className="text-3xl font-bold text-white">{stat.value}</p>
            <p className="text-sm text-gray-400 mt-1">{stat.label}</p>
          </div>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Token Usage Trend */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
          <div className="flex items-center gap-2 mb-6">
            <TrendingUp className="w-5 h-5 text-purple-400" />
            <h2 className="text-lg font-semibold text-white">请求量趋势</h2>
          </div>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={trends}>
              <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
              <XAxis dataKey="date" stroke="#71717a" fontSize={12} />
              <YAxis stroke="#71717a" fontSize={12} />
              <Tooltip
                contentStyle={{
                  background: "#1a1a24",
                  border: "1px solid #27272a",
                  borderRadius: "8px",
                  color: "#fff",
                }}
              />
              <Line type="monotone" dataKey="requests" stroke="#a855f7" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="tokens" stroke="#3b82f6" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Model Distribution */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
          <div className="flex items-center gap-2 mb-6">
            <BarChart3 className="w-5 h-5 text-purple-400" />
            <h2 className="text-lg font-semibold text-white">模型用量分布</h2>
          </div>
          {modelDist.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={modelDist}
                  dataKey="count"
                  nameKey="model"
                  cx="50%"
                  cy="50%"
                  outerRadius={100}
                  innerRadius={60}
                  label={({ model }) => model?.slice(0, 15) || "N/A"}
                >
                  {modelDist.map((_, index) => (
                    <Cell key={index} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    background: "#1a1a24",
                    border: "1px solid #27272a",
                    borderRadius: "8px",
                    color: "#fff",
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-[300px] text-gray-500">
              暂无数据
            </div>
          )}
        </div>
      </div>

      {/* Recent Requests */}
      <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
        <div className="px-6 py-4 border-b border-[#27272a]">
          <h2 className="text-lg font-semibold text-white">最近请求</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-400">
                <th className="px-6 py-3">Request ID</th>
                <th className="px-6 py-3">模型</th>
                <th className="px-6 py-3">延迟</th>
                <th className="px-6 py-3">复杂度</th>
                <th className="px-6 py-3">Tokens</th>
                <th className="px-6 py-3">时间</th>
              </tr>
            </thead>
            <tbody>
              {recentRequests.slice(0, 10).map((req) => (
                <tr key={req.request_id} className="border-t border-[#1a1a24] text-sm">
                  <td className="px-6 py-3 font-mono text-gray-400 text-xs">
                    {req.request_id?.slice(0, 20)}...
                  </td>
                  <td className="px-6 py-3 text-white">{req.model_id || "N/A"}</td>
                  <td className="px-6 py-3 text-gray-300">{req.latency_ms}ms</td>
                  <td className="px-6 py-3">
                    <span className={`px-2 py-1 rounded text-xs ${
                      (req.complexity_score || 0) > 7 ? "bg-red-500/10 text-red-400" :
                      (req.complexity_score || 0) > 4 ? "bg-yellow-500/10 text-yellow-400" :
                      "bg-green-500/10 text-green-400"
                    }`}>
                      {req.complexity_score?.toFixed(1) || "N/A"}
                    </span>
                  </td>
                  <td className="px-6 py-3 text-gray-300">
                    {(req.prompt_tokens || 0) + (req.completion_tokens || 0)}
                  </td>
                  <td className="px-6 py-3 text-gray-500 text-xs">
                    {new Date(req.created_at).toLocaleTimeString("zh-CN")}
                  </td>
                </tr>
              ))}
              {recentRequests.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                    暂无请求记录
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
