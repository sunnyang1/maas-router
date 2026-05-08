import { useEffect, useState } from "react";
import { Activity, Server, AlertTriangle, Clock, Shield, BarChart3 } from "lucide-react";
import { monitoringApi } from "../../services/api";

export default function MonitoringPage() {
  const [services, setServices] = useState<any[]>([]);
  const [metrics, setMetrics] = useState<any>(null);
  const [failoverLogs, setFailoverLogs] = useState<any[]>([]);
  const [alerts, setAlerts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [sv, mt, fl, al] = await Promise.all([
          monitoringApi.services(),
          monitoringApi.metrics(),
          monitoringApi.failoverLogs(),
          monitoringApi.alerts(),
        ]);
        setServices(sv.services || []);
        setMetrics(mt);
        setFailoverLogs(fl.data || []);
        setAlerts(al.data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  if (loading) return <div className="flex items-center justify-center h-96"><div className="animate-spin w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full" /></div>;

  return (
    <div>
      <h1 className="text-2xl font-bold text-white mb-8">运维监控</h1>

      {/* Service Health */}
      <div className="mb-8">
        <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <Server className="w-5 h-5 text-purple-400" />
          服务健康状态
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          {services.map((svc) => (
            <div key={svc.name} className="bg-[#111118] border border-[#27272a] rounded-xl p-4 text-center">
              <div className={`w-3 h-3 rounded-full mx-auto mb-2 ${
                svc.status === "healthy" ? "bg-green-500" : "bg-red-500"
              }`} />
              <p className="text-sm text-white font-medium">{svc.name}</p>
              <p className="text-xs text-gray-500 mt-1">{svc.uptime_pct}%</p>
            </div>
          ))}
        </div>
      </div>

      {/* Real-time Metrics */}
      <div className="mb-8">
        <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <BarChart3 className="w-5 h-5 text-purple-400" />
          实时指标
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          {[
            { label: "QPS", value: metrics?.qps || 0, color: "text-purple-400" },
            { label: "P50 延迟", value: `${metrics?.p50_latency_ms || 0}ms`, color: "text-blue-400" },
            { label: "P99 延迟", value: `${metrics?.p99_latency_ms || 0}ms`, color: "text-yellow-400" },
            { label: "错误率", value: `${metrics?.error_rate_pct || 0}%`, color: "text-red-400" },
            { label: "缓存命中", value: `${metrics?.cache_hit_rate_pct || 0}%`, color: "text-green-400" },
            { label: "自建占比", value: `${metrics?.self_hosted_ratio_pct || 0}%`, color: "text-purple-300" },
          ].map((m) => (
            <div key={m.label} className="bg-[#111118] border border-[#27272a] rounded-xl p-4 text-center">
              <p className={`text-2xl font-bold ${m.color}`}>{m.value}</p>
              <p className="text-xs text-gray-500 mt-1">{m.label}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Alerts & Failover */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Active Alerts */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-[#27272a] flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-yellow-400" />
            <h2 className="text-lg font-semibold text-white">活跃告警</h2>
          </div>
          <div className="p-4 space-y-3">
            {alerts.map((alert, i) => (
              <div key={i} className={`p-3 rounded-lg ${
                alert.level === "warning" ? "bg-yellow-500/5 border border-yellow-500/20" : "bg-blue-500/5 border border-blue-500/20"
              }`}>
                <p className="text-sm text-white">{alert.message}</p>
                <p className="text-xs text-gray-500 mt-1">{alert.time}</p>
              </div>
            ))}
            {alerts.length === 0 && (
              <p className="text-sm text-gray-500 py-4 text-center">暂无活跃告警 ✅</p>
            )}
          </div>
        </div>

        {/* Failover Log */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-[#27272a] flex items-center gap-2">
            <Shield className="w-5 h-5 text-purple-400" />
            <h2 className="text-lg font-semibold text-white">故障切换日志</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-xs text-gray-400 border-b border-[#1a1a24]">
                  <th className="px-4 py-3">时间</th>
                  <th className="px-4 py-3">错误</th>
                  <th className="px-4 py-3">供应商</th>
                  <th className="px-4 py-3">模型</th>
                </tr>
              </thead>
              <tbody>
                {failoverLogs.map((log, i) => (
                  <tr key={i} className="border-t border-[#1a1a24] text-xs">
                    <td className="px-4 py-3 text-gray-500">{new Date(log.time).toLocaleString("zh-CN")}</td>
                    <td className="px-4 py-3 text-red-400">{log.error_code}</td>
                    <td className="px-4 py-3 text-gray-300">{log.provider_id}</td>
                    <td className="px-4 py-3 text-gray-400">{log.model_id}</td>
                  </tr>
                ))}
                {failoverLogs.length === 0 && (
                  <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500 text-sm">暂无故障切换记录</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
