'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Eye, EyeOff, Loader2, ArrowRight, CheckCircle } from 'lucide-react';
import { useUserStore } from '@/stores/userStore';

export default function RegisterPage() {
  const router = useRouter();
  const { register } = useUserStore();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [agreedToTerms, setAgreedToTerms] = useState(false);

  const validatePassword = (pwd: string) => {
    const checks = {
      length: pwd.length >= 8,
      hasUpper: /[A-Z]/.test(pwd),
      hasLower: /[a-z]/.test(pwd),
      hasNumber: /[0-9]/.test(pwd),
    };
    return checks;
  };

  const passwordChecks = validatePassword(password);
  const isPasswordValid = Object.values(passwordChecks).every(Boolean);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // 验证
    if (!isPasswordValid) {
      setError('密码不符合要求');
      return;
    }

    if (password !== confirmPassword) {
      setError('两次输入的密码不一致');
      return;
    }

    if (!agreedToTerms) {
      setError('请阅读并同意服务条款');
      return;
    }

    setIsLoading(true);

    try {
      await register(email, password, name);
      router.push('/dashboard');
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || err.message || '注册失败，请重试';
      setError(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-bg-primary grain flex items-center justify-center px-4 relative overflow-hidden">
      {/* Background Effects */}
      <div className="absolute inset-0 bg-gradient-radial opacity-50" />
      <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-accent/10 rounded-full blur-[120px]" />
      <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-accent-2/10 rounded-full blur-[100px]" />
      
      <div className="relative z-10 w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-10">
          <Link href="/" className="inline-block">
            <span className="font-display text-3xl font-bold gradient-text">MaaS Router</span>
          </Link>
        </div>

        {/* Card */}
        <div className="bg-bg-tertiary/80 backdrop-blur-xl rounded-2xl border border-white/10 p-8 shadow-card">
          <div className="text-center mb-8">
            <h1 className="font-display text-2xl font-bold mb-2">创建账户</h1>
            <p className="text-text-secondary text-body-small">开始使用 MaaS Router</p>
          </div>

          {error && (
            <div className="mb-6 p-4 rounded-xl bg-error/10 border border-error/20 text-error text-sm">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-body-small font-medium mb-2 text-text-secondary">姓名</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="input-field"
                placeholder="您的姓名"
                required
              />
            </div>

            <div>
              <label className="block text-body-small font-medium mb-2 text-text-secondary">邮箱</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="input-field"
                placeholder="your@email.com"
                required
              />
            </div>

            <div>
              <label className="block text-body-small font-medium mb-2 text-text-secondary">密码</label>
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="input-field pr-12"
                  placeholder="••••••••"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-4 top-1/2 -translate-y-1/2 text-text-tertiary hover:text-text-secondary transition-colors"
                >
                  {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                </button>
              </div>
              
              {/* Password requirements */}
              {password && (
                <div className="mt-3 space-y-2">
                  <div className="flex items-center gap-2 text-xs">
                    <CheckCircle className={`w-4 h-4 ${passwordChecks.length ? 'text-success' : 'text-text-muted'}`} />
                    <span className={passwordChecks.length ? 'text-success' : 'text-text-muted'}>
                      至少 8 个字符
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <CheckCircle className={`w-4 h-4 ${passwordChecks.hasUpper ? 'text-success' : 'text-text-muted'}`} />
                    <span className={passwordChecks.hasUpper ? 'text-success' : 'text-text-muted'}>
                      包含大写字母
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <CheckCircle className={`w-4 h-4 ${passwordChecks.hasLower ? 'text-success' : 'text-text-muted'}`} />
                    <span className={passwordChecks.hasLower ? 'text-success' : 'text-text-muted'}>
                      包含小写字母
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <CheckCircle className={`w-4 h-4 ${passwordChecks.hasNumber ? 'text-success' : 'text-text-muted'}`} />
                    <span className={passwordChecks.hasNumber ? 'text-success' : 'text-text-muted'}>
                      包含数字
                    </span>
                  </div>
                </div>
              )}
            </div>

            <div>
              <label className="block text-body-small font-medium mb-2 text-text-secondary">确认密码</label>
              <div className="relative">
                <input
                  type={showConfirmPassword ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="input-field pr-12"
                  placeholder="••••••••"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-4 top-1/2 -translate-y-1/2 text-text-tertiary hover:text-text-secondary transition-colors"
                >
                  {showConfirmPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                </button>
              </div>
              {confirmPassword && password !== confirmPassword && (
                <p className="mt-2 text-xs text-error">密码不一致</p>
              )}
            </div>

            <div className="flex items-start gap-3">
              <input
                type="checkbox"
                id="terms"
                checked={agreedToTerms}
                onChange={(e) => setAgreedToTerms(e.target.checked)}
                className="mt-1 w-4 h-4 rounded border-white/20 bg-bg-tertiary accent-accent"
              />
              <label htmlFor="terms" className="text-body-small text-text-tertiary cursor-pointer">
                我已阅读并同意{' '}
                <Link href="/terms" className="text-accent hover:text-accent-hover">
                  服务条款
                </Link>
                {' '}和{' '}
                <Link href="/privacy" className="text-accent hover:text-accent-hover">
                  隐私政策
                </Link>
              </label>
            </div>

            <button
              type="submit"
              disabled={isLoading || !isPasswordValid}
              className="w-full py-3.5 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all duration-300 hover:shadow-glow disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {isLoading ? (
                <>
                  <Loader2 className="w-5 h-5 animate-spin" />
                  注册中...
                </>
              ) : (
                <>
                  创建账户
                  <ArrowRight className="w-5 h-5" />
                </>
              )}
            </button>
          </form>

          <div className="mt-8 text-center text-body-small">
            <span className="text-text-tertiary">已有账户？</span>{' '}
            <Link href="/login" className="text-accent hover:text-accent-hover font-medium transition-colors">
              立即登录
            </Link>
          </div>
        </div>
        
        {/* Back to home */}
        <div className="text-center mt-6">
          <Link href="/" className="text-text-muted hover:text-text-secondary text-body-small transition-colors">
            ← 返回首页
          </Link>
        </div>
      </div>
    </div>
  );
}
