import { useEffect, useState } from "react";
import { Search, Plus, MoreVertical, Edit, Ban, Key, Coins } from "lucide-react";
import { usersApi } from "../../services/api";

export default function UsersPage() {
  const [users, setUsers] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [selectedUser, setSelectedUser] = useState<any>(null);
  const [showCreate, setShowCreate] = useState(false);

  const [newUser, setNewUser] = useState({ email: "", password: "", display_name: "", plan_id: "free" });

  useEffect(() => {
    loadUsers();
  }, [page, search]);

  async function loadUsers() {
    setLoading(true);
    try {
      const data = await usersApi.list({ page, page_size: 20, search: search || undefined });
      setUsers(data.data || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate() {
    try {
      await usersApi.create(newUser);
      setShowCreate(false);
      setNewUser({ email: "", password: "", display_name: "", plan_id: "free" });
      loadUsers();
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleToggleStatus(userId: string, newStatus: string) {
    try {
      await usersApi.update(userId, { status: newStatus });
      loadUsers();
    } catch (err: any) {
      alert(err.message);
    }
  }

  const planLabels: Record<string, string> = { free: "免费版", pro: "专业版", enterprise: "企业版" };
  const planColors: Record<string, string> = { free: "bg-gray-500/10 text-gray-400", pro: "bg-purple-500/10 text-purple-400", enterprise: "bg-yellow-500/10 text-yellow-400" };

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-2xl font-bold text-white">用户管理</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 bg-purple-600 hover:bg-purple-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          新建用户
        </button>
      </div>

      {/* Search */}
      <div className="bg-[#111118] border border-[#27272a] rounded-xl p-4 mb-6">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-500" />
          <input
            type="text"
            placeholder="搜索邮箱或用户名..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg pl-10 pr-4 py-3 text-white placeholder-gray-600 focus:outline-none focus:border-purple-500"
          />
        </div>
      </div>

      {/* Users Table */}
      <div className="bg-[#111118] border border-[#27272a] rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-400 border-b border-[#27272a]">
                <th className="px-6 py-4">用户</th>
                <th className="px-6 py-4">邮箱</th>
                <th className="px-6 py-4">套餐</th>
                <th className="px-6 py-4">余额</th>
                <th className="px-6 py-4">状态</th>
                <th className="px-6 py-4">注册时间</th>
                <th className="px-6 py-4">操作</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-gray-500">
                    加载中...
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-gray-500">
                    暂无用户数据
                  </td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr key={user.id} className="border-t border-[#1a1a24] text-sm hover:bg-[#1a1a24]/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-full bg-purple-500/20 flex items-center justify-center text-purple-400 font-medium">
                          {(user.display_name || user.email)[0].toUpperCase()}
                        </div>
                        <span className="text-white font-medium">{user.display_name || "未设置"}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-gray-300">{user.email}</td>
                    <td className="px-6 py-4">
                      <span className={`px-2 py-1 rounded text-xs ${planColors[user.plan_id] || ""}`}>
                        {planLabels[user.plan_id] || user.plan_id}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-gray-300">{user.cred_balance?.toFixed(2)} CRED</td>
                    <td className="px-6 py-4">
                      <span className={`px-2 py-1 rounded text-xs ${
                        user.status === "active" ? "bg-green-500/10 text-green-400" : "bg-red-500/10 text-red-400"
                      }`}>
                        {user.status === "active" ? "活跃" : "已禁用"}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-gray-500 text-xs">
                      {user.created_at ? new Date(user.created_at).toLocaleDateString("zh-CN") : "N/A"}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => setSelectedUser(user)}
                          className="p-1.5 rounded hover:bg-[#27272a] text-gray-400 hover:text-white transition-colors"
                        >
                          <Edit className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleToggleStatus(user.id, user.status === "active" ? "suspended" : "active")}
                          className="p-1.5 rounded hover:bg-[#27272a] text-gray-400 hover:text-red-400 transition-colors"
                        >
                          <Ban className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="px-6 py-4 border-t border-[#27272a] flex items-center justify-between">
          <span className="text-sm text-gray-400">
            共 {total} 条记录，第 {page} 页
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="px-3 py-1 rounded border border-[#27272a] text-sm text-gray-400 hover:text-white disabled:opacity-30"
            >
              上一页
            </button>
            <button
              onClick={() => setPage(p => p + 1)}
              disabled={page * 20 >= total}
              className="px-3 py-1 rounded border border-[#27272a] text-sm text-gray-400 hover:text-white disabled:opacity-30"
            >
              下一页
            </button>
          </div>
        </div>
      </div>

      {/* Create User Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-[#111118] border border-[#27272a] rounded-xl p-6 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-white mb-6">新建用户</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">邮箱</label>
                <input type="email" value={newUser.email} onChange={(e) => setNewUser({...newUser, email: e.target.value})} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">密码</label>
                <input type="password" value={newUser.password} onChange={(e) => setNewUser({...newUser, password: e.target.value})} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">用户名</label>
                <input type="text" value={newUser.display_name} onChange={(e) => setNewUser({...newUser, display_name: e.target.value})} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">套餐</label>
                <select value={newUser.plan_id} onChange={(e) => setNewUser({...newUser, plan_id: e.target.value})} className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-3 py-2 text-white text-sm">
                  <option value="free">免费版</option>
                  <option value="pro">专业版</option>
                  <option value="enterprise">企业版</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border border-[#27272a] text-sm text-gray-400 hover:text-white">取消</button>
              <button onClick={handleCreate} className="px-4 py-2 rounded-lg bg-purple-600 hover:bg-purple-700 text-sm font-medium">创建</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
