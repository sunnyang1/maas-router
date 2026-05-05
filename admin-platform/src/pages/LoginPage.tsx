import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Zap } from "lucide-react";
import { authApi, setAuthToken } from "../services/api";

export default function LoginPage() {
  const [email, setEmail] = useState("admin@maas-router.com");
  const [password, setPassword] = useState("admin123");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const data = await authApi.login(email, password);
      setAuthToken(data.access_token);
      navigate("/dashboard");
    } catch (err: any) {
      setError(err.message || "登录失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-[#0a0a0f]">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-2 mb-4">
            <Zap className="w-10 h-10 text-purple-500" />
            <span className="text-3xl font-bold text-white">MaaS-Router</span>
          </div>
          <p className="text-gray-400">管理后台登录</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-[#111118] border border-[#27272a] rounded-xl p-8 space-y-6"
        >
          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm text-gray-400 mb-2">邮箱</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-4 py-3 text-white placeholder-gray-600 focus:outline-none focus:border-purple-500 transition-colors"
              required
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-2">密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-[#0a0a0f] border border-[#27272a] rounded-lg px-4 py-3 text-white placeholder-gray-600 focus:outline-none focus:border-purple-500 transition-colors"
              required
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-purple-600 hover:bg-purple-700 disabled:opacity-50 text-white py-3 rounded-lg font-medium transition-colors"
          >
            {loading ? "登录中..." : "登录"}
          </button>

          <p className="text-xs text-gray-600 text-center">
            演示账号: admin@maas-router.com / admin123
          </p>
        </form>
      </div>
    </div>
  );
}
