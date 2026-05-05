import { useEffect, useState } from "react";
import { DollarSign, TrendingUp, CreditCard, Coins, Download } from "lucide-react";
import { billingApi } from "../../services/api";

export default function BillingPage() {
  const [overview, setOverview] = useState<any>(null);
  const [transactions, setTransactions] = useState<any[]>([]);
  const [credSupply, setCredSupply] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [ov, tx, cs] = await Promise.all([
          billingApi.overview(),
          billingApi.transactions({ page_size: 20 }),
          billingApi.credSupply(),
        ]);
        setOverview(ov);
        setTransactions(tx.data || []);
        setCredSupply(cs);
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
      <h1 className="text-2xl font-bold text-white mb-8">计费管理</h1>

      {/* Revenue Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {[
          { label: "今日用量收入", value: `${(overview?.today_usage_revenue || 0).toFixed(2)} CRED`, icon: DollarSign, color: "text-green-400", bg: "bg-green-500/10" },
          { label: "今日充值", value: `$${(overview?.today_topup || 0).toLocaleString()}`, icon: TrendingUp, color: "text-blue-400", bg: "bg-blue-500/10" },
          { label: "月用量收入", value: `${(overview?.monthly_usage_revenue || 0).toFixed(2)} CRED`, icon: CreditCard, color: "text-purple-400", bg: "bg-purple-500/10" },
          { label: "CRED 流通量", value: `${(overview?.total_cred_circulation || 0).toFixed(0)}`, icon: Coins, color: "text-yellow-400", bg: "bg-yellow-500/10" },
        ].map((s) => (
          <div key={s.label} className="bg-[#111118] border border-[#27272a] rounded-xl p-6">
            <div className={`p-2 rounded-lg ${s.bg} w-fit mb-4`}><s.icon className={`w-5 h-5 ${s.color}`} /></div>
            <p className="text-2xl font-bold text-white">{s.value}</p>
            <p className="text-sm text-gray-400 mt-1">{s.label}</p>
          </div>
        ))}
      </div>

      {/* CRED Supply */}
      <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6 mb-8">
        <h2 className="text-lg font-semibold text-white mb-4">$CRED 供应</h2>
        <div className="grid grid-cols-3 gap-8">
          <div>
            <p className="text-sm text-gray-400 mb-1">总发行量</p>
            <p className="text-2xl font-bold text-white">10,000,000 CRED</p>
          </div>
          <div>
            <p className="text-sm text-gray-400 mb-1">当前流通</p>
            <p className="text-2xl font-bold text-purple-400">{credSupply?.total_supply?.toFixed(0) || 0} CRED</p>
          </div>
          <div>
            <p className="text-sm text-gray-400 mb-1">持有者</p>
            <p className="text-2xl font-bold text-white">{credSupply?.holders || 0} 人</p>
          </div>
        </div>
        <div className="mt-4 p-3 rounded-lg bg-green-500/5 border border-green-500/20">
          <p className="text-sm text-green-400">准备金率: {credSupply?.reserve_ratio || "100%"}</p>
        </div>
      </div>

      {/* Transactions */}
      <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
        <div className="px-6 py-4 border-b border-[#27272a] flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">交易记录</h2>
          <button className="flex items-center gap-2 text-sm text-gray-400 hover:text-white">
            <Download className="w-4 h-4" />
            导出报表
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-400 border-b border-[#27272a]">
                <th className="px-6 py-4">时间</th>
                <th className="px-6 py-4">用户</th>
                <th className="px-6 py-4">类型</th>
                <th className="px-6 py-4">金额</th>
                <th className="px-6 py-4">模型</th>
                <th className="px-6 py-4">Tokens</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((tx) => (
                <tr key={tx.id} className="border-t border-[#1a1a24] text-sm hover:bg-[#1a1a24]/50">
                  <td className="px-6 py-4 text-gray-500 text-xs">
                    {new Date(tx.created_at).toLocaleString("zh-CN")}
                  </td>
                  <td className="px-6 py-4 text-white">{tx.user_email || tx.user_name || "N/A"}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded text-xs ${
                      tx.type === "usage" ? "bg-blue-500/10 text-blue-400" :
                      tx.type === "topup" ? "bg-green-500/10 text-green-400" :
                      "bg-gray-500/10 text-gray-400"
                    }`}>
                      {tx.type === "usage" ? "消费" : tx.type === "topup" ? "充值" : tx.type}
                    </span>
                  </td>
                  <td className={`px-6 py-4 ${tx.amount > 0 ? "text-green-400" : "text-red-400"}`}>
                    {tx.amount > 0 ? "+" : ""}{tx.amount.toFixed(4)} {tx.currency}
                  </td>
                  <td className="px-6 py-4 text-gray-300">{tx.model_id || "N/A"}</td>
                  <td className="px-6 py-4 text-gray-400">{tx.total_tokens?.toLocaleString() || 0}</td>
                </tr>
              ))}
              {transactions.length === 0 && (
                <tr><td colSpan={6} className="px-6 py-8 text-center text-gray-500">暂无交易记录</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
