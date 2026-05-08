'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { 
  Zap, 
  CreditCard, 
  Gift, 
  User,
  Home
} from 'lucide-react';

const navItems = [
  { href: '/dashboard', label: '控制台', icon: Home },
  { href: '/api-keys', label: '密钥', icon: CreditCard },
  { href: '/affiliate', label: '返利', icon: Gift },
  { href: '/redeem', label: '充值', icon: Zap },
  { href: '/profile', label: '我的', icon: User },
];

export function BottomNav() {
  const pathname = usePathname();

  const isActive = (href: string) => pathname === href || pathname.startsWith(href + '/');

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 glass border-t border-white/5 md:hidden safe-area-bottom">
      <div className="flex items-center justify-around h-16">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = isActive(item.href);
          
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex flex-col items-center justify-center flex-1 h-full transition-colors ${
                active ? 'text-accent' : 'text-text-tertiary'
              }`}
            >
              <div className={`relative p-1.5 rounded-lg transition-colors ${
                active ? 'bg-accent/10' : ''
              }`}>
                <Icon className="w-5 h-5" />
                {active && (
                  <span className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-1 h-1 rounded-full bg-accent" />
                )}
              </div>
              <span className={`text-xs mt-0.5 ${active ? 'font-medium' : ''}`}>
                {item.label}
              </span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
