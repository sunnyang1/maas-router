import { useEffect, useState } from "react";
import { Cpu, Server, Route, Plus, Edit, Trash2 } from "lucide-react";
import { modelsApi } from "../../services/api";

export default function ModelsPage() {
  const [tab, setTab] = useState<"providers" | "models" | "routing">("providers");
  const [providers, setProviders] = useState<any[]>([]);
  const [models, setModels] = useState<any[]>([]);
  const [rules, setRules] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [tab]);

  async function loadData() {
    setLoading(true);
    try {
      if (tab === "providers") {
        const data = await modelsApi.listProviders();
        setProviders(data.data || []);
      } else if (tab === "models") {
        const data = await modelsApi.listModels();
        setModels(data.data || []);
      } else {
        const data = await modelsApi.listRoutingRules();
        setRules(data.data || []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  async function toggleProvider(id: string, currentStatus: string) {
    await modelsApi.toggleProvider(id, currentStatus === "active" ? "offline" : "active");
    loadData();
  }

  async function toggleModel(id: string, currentStatus: string) {
    await modelsApi.toggleModel(id, currentStatus === "active" ? "inactive" : "active");
    loadData();
  }

  async function deleteRule(id: string) {
    await modelsApi.deleteRoutingRule(id);
    loadData();
  }

  const tabs = [
    { id: "providers" as const, label: "供应商", icon: Server },
    { id: "models" as const, label: "模型", icon: Cpu },
    { id: "routing" as const, label: "路由规则", icon: Route },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-white mb-8">模型管理</h1>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              tab === t.id ? "bg-purple-600 text-white" : "bg-[#111118] border border-[#27272a] text-gray-400 hover:text-white"
            }`}
          >
            <t.icon className="w-4 h-4" />
            {t.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full" />
        </div>
      ) : (
        <>
          {/* Providers */}
          {tab === "providers" && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {providers.map((p) => (
                <div key={p.id} className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center">
                        <Server className="w-5 h-5 text-purple-400" />
                      </div>
                      <div>
                        <h3 className="text-white font-semibold">{p.name}</h3>
                        <p className="text-xs text-gray-500">{p.id}</p>
                      </div>
                    </div>
                    <span className={`px-2 py-1 rounded text-xs ${
                      p.status === "active" ? "bg-green-500/10 text-green-400" : "bg-red-500/10 text-red-400"
                    }`}>
                      {p.status === "active" ? "在线" : "离线"}
                    </span>
                  </div>
                  <p className="text-sm text-gray-400 mb-4">{p.description}</p>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500">{p.model_count} 个模型</span>
                    <button
                      onClick={() => toggleProvider(p.id, p.status)}
                      className={`text-xs px-3 py-1 rounded ${
                        p.status === "active"
                          ? "border border-red-500/30 text-red-400 hover:bg-red-500/10"
                          : "border border-green-500/30 text-green-400 hover:bg-green-500/10"
                      }`}
                    >
                      {p.status === "active" ? "禁用" : "启用"}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Models */}
          {tab === "models" && (
            <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="text-left text-sm text-gray-400 border-b border-[#27272a]">
                      <th className="px-6 py-4">模型</th>
                      <th className="px-6 py-4">供应商</th>
                      <th className="px-6 py-4">价格 (输入/输出)</th>
                      <th className="px-6 py-4">上下文</th>
                      <th className="px-6 py-4">状态</th>
                      <th className="px-6 py-4">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {models.map((m) => (
                      <tr key={m.id} className="border-t border-[#1a1a24] text-sm hover:bg-[#1a1a24]/50">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <Cpu className="w-4 h-4 text-purple-400" />
                            <span className="text-white font-medium">{m.name}</span>
                            {m.is_recommended && (
                              <span className="px-1.5 py-0.5 rounded bg-purple-500/10 text-purple-400 text-xs">推荐</span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 text-gray-300">{m.provider_name || m.provider_id}</td>
                        <td className="px-6 py-4 text-gray-300">
                          ${m.input_price} / ${m.output_price}
                        </td>
                        <td className="px-6 py-4 text-gray-300">
                          {m.context_window ? `${(m.context_window / 1000).toFixed(0)}K` : "N/A"}
                        </td>
                        <td className="px-6 py-4">
                          <span className={`px-2 py-1 rounded text-xs ${
                            m.status === "active" ? "bg-green-500/10 text-green-400" : "bg-red-500/10 text-red-400"
                          }`}>
                            {m.status === "active" ? "活跃" : "禁用"}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <button
                            onClick={() => toggleModel(m.id, m.status)}
                            className="text-xs px-3 py-1 rounded border border-[#27272a] text-gray-400 hover:text-white"
                          >
                            {m.status === "active" ? "禁用" : "启用"}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Routing Rules */}
          {tab === "routing" && (
            <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
              <div className="px-6 py-4 border-b border-[#27272a] flex items-center justify-between">
                <h3 className="text-white font-semibold">路由规则列表</h3>
                <button className="flex items-center gap-2 bg-purple-600 hover:bg-purple-700 px-4 py-2 rounded-lg text-sm">
                  <Plus className="w-4 h-4" />
                  添加规则
                </button>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="text-left text-sm text-gray-400 border-b border-[#27272a]">
                      <th className="px-6 py-4">优先级</th>
                      <th className="px-6 py-4">规则名称</th>
                      <th className="px-6 py-4">条件</th>
                      <th className="px-6 py-4">目标</th>
                      <th className="px-6 py-4">状态</th>
                      <th className="px-6 py-4">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rules.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                          暂无路由规则。默认使用复杂度评分自动路由。
                        </td>
                      </tr>
                    ) : (
                      rules.map((r) => (
                        <tr key={r.id} className="border-t border-[#1a1a24] text-sm">
                          <td className="px-6 py-4 text-purple-400 font-mono">{r.priority}</td>
                          <td className="px-6 py-4 text-white">{r.name}</td>
                          <td className="px-6 py-4 text-gray-400 text-xs">
                            <pre className="max-w-xs truncate">{JSON.stringify(r.condition)}</pre>
                          </td>
                          <td className="px-6 py-4 text-gray-400 text-xs">
                            <pre className="max-w-xs truncate">{JSON.stringify(r.action)}</pre>
                          </td>
                          <td className="px-6 py-4">
                            <span className="px-2 py-1 rounded text-xs bg-green-500/10 text-green-400">活跃</span>
                          </td>
                          <td className="px-6 py-4">
                            <button
                              onClick={() => deleteRule(r.id)}
                              className="p-1.5 rounded hover:bg-red-500/10 text-gray-400 hover:text-red-400"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
