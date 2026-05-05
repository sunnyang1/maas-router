import { Outlet, NavLink, useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  Users,
  Cpu,
  DollarSign,
  Activity,
  Settings,
  LogOut,
  Zap,
} from "lucide-react";
import { setAuthToken } from "../../services/api";

const navItems = [
  { to: "/dashboard", icon: LayoutDashboard, label: "概览" },
  { to: "/users", icon: Users, label: "用户" },
  { to: "/models", icon: Cpu, label: "模型" },
  { to: "/billing", icon: DollarSign, label: "计费" },
  { to: "/monitoring", icon: Activity, label: "运维" },
  { to: "/settings", icon: Settings, label: "设置" },
];

export default function AdminLayout() {
  const navigate = useNavigate();

  const handleLogout = () => {
    setAuthToken(null);
    navigate("/login");
  };

  return (
    <div className="flex h-screen bg-[#0a0a0f]">
      {/* Sidebar */}
      <aside className="w-64 bg-[#111118] border-r border-[#27272a] flex flex-col">
        <div className="p-6 border-b border-[#27272a]">
          <div className="flex items-center gap-2">
            <Zap className="w-6 h-6 text-purple-500" />
            <span className="text-lg font-bold text-white">MaaS-Router</span>
          </div>
          <p className="text-xs text-gray-500 mt-1">管理后台</p>
        </div>

        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-4 py-3 rounded-lg text-sm transition-colors ${
                  isActive
                    ? "bg-purple-500/10 text-purple-400"
                    : "text-gray-400 hover:text-white hover:bg-[#1a1a24]"
                }`
              }
            >
              <item.icon className="w-5 h-5" />
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="p-4 border-t border-[#27272a]">
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 px-4 py-3 w-full rounded-lg text-sm text-gray-400 hover:text-red-400 hover:bg-[#1a1a24] transition-colors"
          >
            <LogOut className="w-5 h-5" />
            退出登录
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
