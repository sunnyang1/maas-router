'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Menu, X, Zap, User, CreditCard, Gift, LogOut } from 'lucide-react';
import { getBrandingSettings, type BrandingSettings } from '@/lib/api/branding';

const navItems = [
  { href: '/dashboard', label: '控制台', icon: Zap },
  { href: '/api-keys', label: 'API密钥', icon: CreditCard },
  { href: '/affiliate', label: '邀请返利', icon: Gift },
  { href: '/redeem', label: '卡密充值', icon: CreditCard },
  { href: '/profile', label: '个人中心', icon: User },
];

export function Header() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const pathname = usePathname();
  const [branding, setBranding] = useState<BrandingSettings | null>(null);

  const isActive = (href: string) => pathname === href || pathname.startsWith(href + '/');

  useEffect(() => {
    getBrandingSettings()
      .then((settings) => {
        setBranding(settings);
      })
      .catch(() => {
        // Silently fail - use defaults
      });
  }, []);

  const isValidUrl = (url: string) => {
    try {
      const parsed = new URL(url);
      return ['https:', 'http:'].includes(parsed.protocol);
    } catch {
      return false;
    }
  };

  const siteName = branding?.site_name || 'MaaS Router';

  return (
    <>
      {/* Announcement Bar */}
      {branding?.announcement && branding.announcement.length <= 500 && (
        <div
          className="text-center text-sm py-2 px-4"
          style={{
            backgroundColor: branding.primary_color || '#1677ff',
            color: '#fff',
          }}
        >
          {branding.announcement}
        </div>
      )}

      <header className="sticky top-0 z-50 glass border-b border-white/5">
        <div className="container-custom">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <Link href="/" className="flex items-center gap-2">
              {branding?.logo_url && isValidUrl(branding.logo_url) ? (
                <img
                  src={branding.logo_url}
                  alt={siteName}
                  className="h-8 w-auto"
                />
              ) : (
                <span
                  className="font-display text-xl font-bold gradient-text"
                  style={
                    branding?.primary_color
                      ? { background: `linear-gradient(135deg, ${branding.primary_color}, ${branding.primary_color}88)`, WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }
                      : undefined
                  }
                >
                  {siteName}
                </span>
              )}
            </Link>

            {/* Desktop Navigation */}
            <nav className="hidden md:flex items-center gap-1">
              {navItems.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    isActive(item.href)
                      ? 'text-accent bg-accent/10'
                      : 'text-text-secondary hover:text-text-primary hover:bg-white/5'
                  }`}
                >
                  {item.label}
                </Link>
              ))}
            </nav>

            {/* Desktop Actions */}
            <div className="hidden md:flex items-center gap-3">
              <button className="flex items-center gap-2 px-4 py-2 text-sm text-text-secondary hover:text-text-primary transition-colors">
                <LogOut className="w-4 h-4" />
                退出
              </button>
            </div>

            {/* Mobile Menu Button */}
            <button
              onClick={() => setIsMenuOpen(!isMenuOpen)}
              className="md:hidden p-2 rounded-lg text-text-secondary hover:text-text-primary hover:bg-white/5 transition-colors"
            >
              {isMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>

          {/* Mobile Navigation */}
          {isMenuOpen && (
            <nav className="md:hidden py-4 border-t border-white/5">
              <div className="space-y-1">
                {navItems.map((item) => {
                  const Icon = item.icon;
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={() => setIsMenuOpen(false)}
                      className={`flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors ${
                        isActive(item.href)
                          ? 'text-accent bg-accent/10'
                          : 'text-text-secondary hover:text-text-primary hover:bg-white/5'
                      }`}
                    >
                      <Icon className="w-5 h-5" />
                      {item.label}
                    </Link>
                  );
                })}
                <button className="w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-white/5 transition-colors">
                  <LogOut className="w-5 h-5" />
                  退出登录
                </button>
              </div>
            </nav>
          )}
        </div>
      </header>
    </>
  );
}
