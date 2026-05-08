'use client';

import { useEffect, useCallback } from 'react';
import Link from 'next/link';
import { useRouter, usePathname } from 'next/navigation';
import { LayoutDashboard, Play, Key, BarChart3, CreditCard, Settings, LogOut, Wallet, Loader2 } from 'lucide-react';
import { useUserStore } from '@/stores/userStore';
import { cn } from '@/lib/utils/cn';

const navItems = [
  { href: '/dashboard', label: '仪表盘', icon: LayoutDashboard },
  { href: '/playground', label: 'Playground', icon: Play },
  { href: '/api-keys', label: 'API Keys', icon: Key },
  { href: '/usage', label: '使用量', icon: BarChart3 },
  { href: '/billing', label: '计费', icon: CreditCard },
  { href: '/settings', label: '设置', icon: Settings },
];

export function DashboardHeader() {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout, isLoading, checkAuth, isAuthenticated } = useUserStore();

  // 检查认证状态
  useEffect(() => {
    // 只在客户端执行
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('access-token') || localStorage.getItem('refresh-token');
      if (token && !isAuthenticated) {
        checkAuth();
      }
    }
  }, [checkAuth, isAuthenticated]);

  // 定期刷新用户信息 (每5分钟)
  useEffect(() => {
    if (!isAuthenticated) return;

    const interval = setInterval(() => {
      useUserStore.getState().refreshUser().catch(() => {
        // Token 刷新失败，API client 会自动处理登出
      });
    }, 5 * 60 * 1000);

    return () => clearInterval(interval);
  }, [isAuthenticated]);

  const handleLogout = useCallback(async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Logout error:', error);
    }
    router.push('/');
  }, [logout, router]);

  // 显示加载状态
  if (isLoading && !isAuthenticated) {
    return (
      <header className="sticky top-0 z-50 bg-bg-primary/80 backdrop-blur-xl border-b border-white/5">
        <div className="container-custom">
          <div className="flex items-center justify-center h-16">
            <Loader2 className="w-6 h-6 text-accent animate-spin" />
          </div>
        </div>
      </header>
    );
  }

  return (
    <header className="sticky top-0 z-50 bg-bg-primary/80 backdrop-blur-xl border-b border-white/5">
      <div className="container-custom">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link href="/" className="font-display text-xl font-bold gradient-text">
            MaaS Router
          </Link>

          {/* Navigation */}
          <nav className="hidden md:flex items-center gap-1">
            {navItems.map((item) => {
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all duration-200",
                    isActive 
                      ? "bg-accent/10 text-accent" 
                      : "text-text-tertiary hover:text-text-primary hover:bg-white/5"
                  )}
                >
                  <item.icon className="w-4 h-4" />
                  {item.label}
                </Link>
              );
            })}
          </nav>

          {/* User Menu */}
          <div className="flex items-center gap-4">
            {/* Balance */}
            <div className="flex items-center gap-2 px-4 py-2 rounded-xl bg-accent/10 border border-accent/20">
              <Wallet className="w-4 h-4 text-accent" />
              <span className="text-sm font-semibold text-accent">
                {user?.credBalance?.toFixed(2) || '0.00'} CRED
              </span>
            </div>
            
            {/* User Info & Logout */}
            <div className="flex items-center gap-3">
              <div className="hidden sm:block">
                <span className="text-sm text-text-tertiary">
                  {user?.email}
                </span>
                {user?.tier && (
                  <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-medium bg-accent-2/10 text-accent-2">
                    {user.tier === 'free' ? '免费版' : user.tier === 'pro' ? '专业版' : '企业版'}
                  </span>
                )}
              </div>
              <button
                onClick={handleLogout}
                className="p-2 rounded-xl text-text-tertiary hover:text-text-primary hover:bg-white/5 transition-all"
                title="退出登录"
              >
                <LogOut className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
