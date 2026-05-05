import { useEffect, useState } from "react";
import { Settings, Shield, FileText, Users } from "lucide-react";
import { settingsApi } from "../../services/api";

export default function SettingsPage() {
  const [config, setConfig] = useState<any>(null);
  const [admins, setAdmins] = useState<any[]>([]);
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [cfg, adm, al] = await Promise.all([
          settingsApi.getConfig(),
          settingsApi.listAdmins(),
          settingsApi.auditLogs(),
        ]);
        setConfig(cfg);
        setAdmins(adm.data || []);
        setLogs(al.data || []);
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
      <h1 className="text-2xl font-bold text-white mb-8">系统设置</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Rate Limit Config */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Shield className="w-5 h-5 text-purple-400" />
            速率限制配置
          </h2>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs text-gray-400 mb-1">免费版 RPM</label>
                <input type="number" defaultValue={config?.rate_limit?.free_rpm || 100} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">免费版 TPM</label>
                <input type="number" defaultValue={config?.rate_limit?.free_tpm || 10000} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">专业版 RPM</label>
                <input type="number" defaultValue={config?.rate_limit?.pro_rpm || 1000} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">专业版 TPM</label>
                <input type="number" defaultValue={config?.rate_limit?.pro_tpm || 100000} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
            </div>
          </div>
        </div>

        {/* Router Config */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Settings className="w-5 h-5 text-purple-400" />
            路由配置
          </h2>
          <div className="space-y-4">
            <div>
              <label className="block text-xs text-gray-400 mb-1">路由策略</label>
              <select defaultValue={config?.router?.strategy || "cost_optimized"} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm">
                <option value="cost_optimized">成本优先</option>
                <option value="performance">性能优先</option>
                <option value="balanced">均衡模式</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-gray-400 mb-1">复杂度阈值</label>
              <input type="number" defaultValue={config?.router?.complexity_threshold || 5} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
            </div>
            <div>
              <label className="block text-xs text-gray-400 mb-1">缓存 TTL (秒)</label>
              <input type="number" defaultValue={config?.router?.cache_ttl_seconds || 3600} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
            </div>
          </div>
        </div>

        {/* Pricing Config */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <FileText className="w-5 h-5 text-purple-400" />
            定价配置
          </h2>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs text-gray-400 mb-1">自建输入价格 ($/M)</label>
                <input type="number" step="0.01" defaultValue={config?.pricing?.self_hosted_input || 0.15} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">自建输出价格 ($/M)</label>
                <input type="number" step="0.01" defaultValue={config?.pricing?.self_hosted_output || 0.50} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">抽成比例 (%)</label>
                <input type="number" defaultValue={config?.pricing?.take_rate_pct || 3} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">CRED 折扣 (%)</label>
                <input type="number" defaultValue={config?.pricing?.cred_discount_pct || 30} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
            </div>
          </div>
        </div>

        {/* Settlement Config */}
        <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Shield className="w-5 h-5 text-purple-400" />
            结算配置
          </h2>
          <div className="space-y-4">
            <div>
              <label className="block text-xs text-gray-400 mb-1">结算时间 (UTC)</label>
              <input type="text" defaultValue={config?.settlement?.time_utc || "00:00"} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
            </div>
            <div>
              <label className="block text-xs text-gray-400 mb-1">L2 网络</label>
              <select defaultValue={config?.settlement?.l2_network || "Polygon"} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm">
                <option value="Polygon">Polygon</option>
                <option value="Arbitrum">Arbitrum</option>
                <option value="Optimism">Optimism</option>
                <option value="Base">Base</option>
              </select>
            </div>
          </div>
        </div>
      </div>
      
      {/* Save button */}
      <div className="mt-6 flex justify-end">
        <button className="bg-purple-600 hover:bg-purple-700 px-6 py-2 rounded-lg text-sm font-medium transition-colors">
          保存配置
        </button>
      </div>

      {/* Audit Logs */}
      <div className="mt-8 bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
        <div className="px-6 py-4 border-b border-[#27272a] flex items-center gap-2">
          <FileText className="w-5 h-5 text-purple-400" />
          <h2 className="text-lg font-semibold text-white">操作审计日志</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-xs text-gray-400 border-b border-[#1a1a24]">
                <th className="px-6 py-3">时间</th>
                <th className="px-6 py-3">操作</th>
                <th className="px-6 py-3">资源类型</th>
                <th className="px-6 py-3">资源 ID</th>
                <th className="px-6 py-3">IP</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id} className="border-t border-[#1a1a24] text-xs">
                  <td className="px-6 py-3 text-gray-500">{new Date(log.created_at).toLocaleString("zh-CN")}</td>
                  <td className="px-6 py-3 text-white">{log.action}</td>
                  <td className="px-6 py-3 text-gray-300">{log.resource_type}</td>
                  <td className="px-6 py-3 text-gray-400 font-mono">{log.resource_id?.slice(0, 12)}...</td>
                  <td className="px-6 py-3 text-gray-500">{log.ip_address}</td>
                </tr>
              ))}
              {logs.length === 0 && (
                <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-500">暂无审计日志</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
