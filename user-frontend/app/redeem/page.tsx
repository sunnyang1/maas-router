'use client';

import { useState, useEffect } from 'react';
import { 
  Gift, 
  CheckCircle, 
  AlertCircle, 
  Loader2,
  Copy,
  History,
  CreditCard,
  Ticket,
  Clock
} from 'lucide-react';
import { apiClient } from '@/lib/api/client';

// 充值记录类型
interface RedeemRecord {
  id: number;
  code: string;
  amount: number;
  redeemed_at: string;
}

export default function RedeemPage() {
  const [code, setCode] = useState('');
  const [isRedeeming, setIsRedeeming] = useState(false);
  const [result, setResult] = useState<{
    success: boolean;
    message: string;
    amount?: number;
    balance?: number;
  } | null>(null);
  const [records, setRecords] = useState<RedeemRecord[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchRecords();
  }, []);

  const fetchRecords = async () => {
    try {
      setIsLoading(true);
      const response = await apiClient.client.get('/user/redeem/records?page=1&page_size=10');
      setRecords(response.data.data || []);
    } catch (error) {
      console.error('获取充值记录失败:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRedeem = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!code.trim()) return;

    try {
      setIsRedeeming(true);
      setResult(null);

      const response = await apiClient.client.post('/user/redeem', {
        code: code.trim().toUpperCase()
      });

      setResult({
        success: true,
        message: '充值成功！',
        amount: response.data.amount,
        balance: response.data.balance
      });

      setCode('');
      fetchRecords(); // 刷新记录
    } catch (error: any) {
      setResult({
        success: false,
        message: error.response?.data?.error?.message || '充值失败，请检查卡密是否正确'
      });
    } finally {
      setIsRedeeming(false);
    }
  };

  const copyCode = (code: string) => {
    navigator.clipboard.writeText(code);
  };

  return (
    <div className="min-h-screen bg-bg-primary">
      {/* 头部 */}
      <div className="bg-bg-secondary border-b border-white/5">
        <div className="container-custom py-6 sm:py-8">
          <h1 className="font-display text-2xl sm:text-3xl font-bold text-text-primary mb-2">
            卡密充值
          </h1>
          <p className="text-text-secondary">
            输入卡密充值码，快速为您的账户充值
          </p>
        </div>
      </div>

      <div className="container-custom py-6 sm:py-8">
        <div className="grid lg:grid-cols-2 gap-6">
          {/* 左侧：充值表单 */}
          <div className="space-y-6">
            {/* 充值卡片 */}
            <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-8">
              <div className="flex items-center gap-3 mb-6">
                <div className="w-12 h-12 rounded-xl bg-accent/10 flex items-center justify-center">
                  <Gift className="w-6 h-6 text-accent" />
                </div>
                <div>
                  <h2 className="font-display text-xl font-semibold text-text-primary">
                    输入卡密
                  </h2>
                  <p className="text-sm text-text-tertiary">
                    请输入您获得的充值卡密
                  </p>
                </div>
              </div>

              <form onSubmit={handleRedeem} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-2">
                    卡密代码
                  </label>
                  <div className="relative">
                    <Ticket className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-text-muted" />
                    <input
                      type="text"
                      value={code}
                      onChange={(e) => setCode(e.target.value.toUpperCase())}
                      placeholder="请输入卡密，如: MRXXXXXXXXXXXXXX"
                      disabled={isRedeeming}
                      className="w-full pl-12 pr-4 py-4 bg-bg-tertiary border border-white/10 rounded-xl text-text-primary placeholder-text-muted focus:outline-none focus:border-accent/50 focus:ring-1 focus:ring-accent/50 transition-colors disabled:opacity-50 font-mono text-lg tracking-wider"
                    />
                  </div>
                  <p className="mt-2 text-xs text-text-tertiary">
                    卡密区分大小写，请准确输入
                  </p>
                </div>

                {/* 结果提示 */}
                {result && (
                  <div className={`p-4 rounded-xl flex items-start gap-3 ${
                    result.success 
                      ? 'bg-green-500/10 border border-green-500/30' 
                      : 'bg-red-500/10 border border-red-500/30'
                  }`}>
                    {result.success ? (
                      <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                    ) : (
                      <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
                    )}
                    <div>
                      <p className={result.success ? 'text-green-400' : 'text-red-400'}>
                        {result.message}
                      </p>
                      {result.success && result.amount && (
                        <p className="text-text-secondary text-sm mt-1">
                          充值金额: <span className="text-accent font-semibold">¥{result.amount.toFixed(2)}</span>
                          {result.balance && (
                            <span className="ml-2">
                              当前余额: <span className="text-accent font-semibold">¥{result.balance.toFixed(2)}</span>
                            </span>
                          )}
                        </p>
                      )}
                    </div>
                  </div>
                )}

                <button
                  type="submit"
                  disabled={isRedeeming || !code.trim()}
                  className="w-full flex items-center justify-center gap-2 px-6 py-4 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all duration-300 hover:shadow-glow disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isRedeeming ? (
                    <>
                      <Loader2 className="w-5 h-5 animate-spin" />
                      充值处理中...
                    </>
                  ) : (
                    <>
                      <CreditCard className="w-5 h-5" />
                      立即充值
                    </>
                  )}
                </button>
              </form>
            </div>

            {/* 使用说明 */}
            <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
              <h3 className="font-display font-semibold text-text-primary mb-4">
                使用说明
              </h3>
              <div className="space-y-3 text-sm text-text-secondary">
                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-accent/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                    <span className="text-accent text-xs font-medium">1</span>
                  </div>
                  <p>获取充值卡密，卡密通常由管理员发放或购买获得</p>
                </div>
                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-accent/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                    <span className="text-accent text-xs font-medium">2</span>
                  </div>
                  <p>在上方输入框中准确输入卡密代码（区分大小写）</p>
                </div>
                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-accent/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                    <span className="text-accent text-xs font-medium">3</span>
                  </div>
                  <p>点击"立即充值"按钮，系统将自动验证并充值到账</p>
                </div>
                <div className="flex items-start gap-3">
                  <div className="w-6 h-6 rounded-full bg-accent/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                    <span className="text-accent text-xs font-medium">4</span>
                  </div>
                  <p>充值成功后，金额将立即添加到您的账户余额中</p>
                </div>
              </div>

              <div className="mt-4 p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg">
                <p className="text-xs text-yellow-400">
                  <AlertCircle className="w-4 h-4 inline mr-1" />
                  注意：卡密为一次性使用，充值成功后即失效，请妥善保管您的卡密。
                </p>
              </div>
            </div>
          </div>

          {/* 右侧：充值记录 */}
          <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="w-10 h-10 rounded-xl bg-accent-2/10 flex items-center justify-center">
                <History className="w-5 h-5 text-accent-2" />
              </div>
              <div>
                <h2 className="font-display text-lg font-semibold text-text-primary">
                  充值记录
                </h2>
                <p className="text-sm text-text-tertiary">
                  最近10条充值记录
                </p>
              </div>
            </div>

            {isLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="w-6 h-6 text-accent animate-spin" />
              </div>
            ) : records.length === 0 ? (
              <div className="text-center py-12 text-text-tertiary">
                <Ticket className="w-16 h-16 mx-auto mb-4 opacity-30" />
                <p className="text-lg">暂无充值记录</p>
                <p className="text-sm mt-2">使用卡密充值后将在此显示记录</p>
              </div>
            ) : (
              <div className="space-y-3">
                {records.map((record) => (
                  <div
                    key={record.id}
                    className="flex items-center justify-between p-4 bg-bg-tertiary/50 rounded-xl"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-green-500/10 flex items-center justify-center">
                        <CheckCircle className="w-5 h-5 text-green-500" />
                      </div>
                      <div>
                        <p className="text-text-primary font-medium">
                          卡密充值
                        </p>
                        <div className="flex items-center gap-2 text-sm text-text-tertiary">
                          <span className="font-mono">{record.code}</span>
                          <button
                            onClick={() => copyCode(record.code)}
                            className="text-accent hover:text-accent-hover"
                          >
                            <Copy className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-accent font-semibold">
                        +¥{record.amount.toFixed(2)}
                      </p>
                      <p className="text-xs text-text-tertiary flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {new Date(record.redeemed_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* 统计信息 */}
            {!isLoading && records.length > 0 && (
              <div className="mt-6 pt-6 border-t border-white/5">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-text-secondary">累计充值次数</span>
                  <span className="text-text-primary font-medium">{records.length} 次</span>
                </div>
                <div className="flex items-center justify-between text-sm mt-2">
                  <span className="text-text-secondary">累计充值金额</span>
                  <span className="text-accent font-semibold">
                    ¥{records.reduce((sum, r) => sum + r.amount, 0).toFixed(2)}
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
