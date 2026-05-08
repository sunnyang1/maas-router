'use client';

import { useState, useEffect } from 'react';
import { 
  Copy, 
  Users, 
  DollarSign, 
  Gift, 
  TrendingUp,
  Wallet,
  ChevronRight,
  Loader2,
  CheckCircle,
  AlertCircle
} from 'lucide-react';
import { apiClient } from '@/lib/api/client';

// 邀请信息类型
interface InviteInfo {
  invite_code: string;
  invite_link: string;
  invite_count: number;
  affiliate_balance: number;
  total_earnings: number;
  min_withdrawal: number;
  recharge_rate: number;
  consumption_rate: number;
  register_reward: number;
}

// 邀请记录类型
interface InviteRecord {
  user_id: number;
  user_name: string;
  email: string;
  created_at: string;
}

// 返利记录类型
interface RebateRecord {
  id: number;
  from_user_id: number;
  type: string;
  amount: number;
  description: string;
  created_at: string;
}

export default function AffiliatePage() {
  const [inviteInfo, setInviteInfo] = useState<InviteInfo | null>(null);
  const [inviteRecords, setInviteRecords] = useState<InviteRecord[]>([]);
  const [rebateRecords, setRebateRecords] = useState<RebateRecord[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [copied, setCopied] = useState(false);
  const [withdrawAmount, setWithdrawAmount] = useState('');
  const [isWithdrawing, setIsWithdrawing] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'invites' | 'records'>('overview');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setIsLoading(true);
      // 并行获取数据
      const [infoRes, invitesRes, recordsRes] = await Promise.all([
        apiClient.client.get('/user/affiliate/info'),
        apiClient.client.get('/user/affiliate/invites?page=1&page_size=5'),
        apiClient.client.get('/user/affiliate/records?page=1&page_size=5'),
      ]);

      setInviteInfo(infoRes.data);
      setInviteRecords(invitesRes.data.data || []);
      setRebateRecords(recordsRes.data.data || []);
    } catch (error) {
      console.error('获取邀请返利数据失败:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleWithdraw = async () => {
    if (!withdrawAmount || parseFloat(withdrawAmount) <= 0) return;
    
    try {
      setIsWithdrawing(true);
      await apiClient.client.post('/user/affiliate/withdraw', {
        amount: parseFloat(withdrawAmount)
      });
      setWithdrawAmount('');
      fetchData(); // 刷新数据
      alert('提现成功！');
    } catch (error: any) {
      alert(error.response?.data?.error?.message || '提现失败');
    } finally {
      setIsWithdrawing(false);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-accent animate-spin" />
      </div>
    );
  }

  if (!inviteInfo) {
    return (
      <div className="min-h-screen bg-bg-primary flex items-center justify-center">
        <div className="text-text-secondary">加载失败，请刷新重试</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-bg-primary">
      {/* 头部 */}
      <div className="bg-bg-secondary border-b border-white/5">
        <div className="container-custom py-6 sm:py-8">
          <h1 className="font-display text-2xl sm:text-3xl font-bold text-text-primary mb-2">
            邀请返利
          </h1>
          <p className="text-text-secondary">
            邀请好友注册，获得丰厚返利奖励
          </p>
        </div>
      </div>

      <div className="container-custom py-6 sm:py-8">
        {/* 统计卡片 */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <StatCard
            icon={<DollarSign className="w-5 h-5" />}
            label="返利余额"
            value={`¥${inviteInfo.affiliate_balance.toFixed(2)}`}
            color="accent"
          />
          <StatCard
            icon={<TrendingUp className="w-5 h-5" />}
            label="累计收益"
            value={`¥${inviteInfo.total_earnings.toFixed(2)}`}
            color="accent-2"
          />
          <StatCard
            icon={<Users className="w-5 h-5" />}
            label="成功邀请"
            value={inviteInfo.invite_count.toString()}
            color="blue"
          />
          <StatCard
            icon={<Gift className="w-5 h-5" />}
            label="注册奖励"
            value={`¥${inviteInfo.register_reward.toFixed(2)}`}
            color="purple"
          />
        </div>

        <div className="grid lg:grid-cols-3 gap-6">
          {/* 左侧：邀请码和提现 */}
          <div className="lg:col-span-1 space-y-6">
            {/* 邀请码卡片 */}
            <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
              <h2 className="font-display text-lg font-semibold text-text-primary mb-4">
                我的邀请码
              </h2>
              
              <div className="bg-bg-tertiary rounded-xl p-4 mb-4">
                <div className="text-2xl sm:text-3xl font-display font-bold text-accent text-center tracking-wider">
                  {inviteInfo.invite_code}
                </div>
              </div>

              <button
                onClick={() => copyToClipboard(inviteInfo.invite_link)}
                className="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-accent/10 border border-accent/30 text-accent hover:bg-accent/20 transition-colors"
              >
                {copied ? (
                  <>
                    <CheckCircle className="w-4 h-4" />
                    已复制
                  </>
                ) : (
                  <>
                    <Copy className="w-4 h-4" />
                    复制邀请链接
                  </>
                )}
              </button>

              <div className="mt-4 p-3 bg-bg-tertiary/50 rounded-lg">
                <p className="text-xs text-text-tertiary">
                  邀请链接: {inviteInfo.invite_link}
                </p>
              </div>
            </div>

            {/* 提现卡片 */}
            <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
              <h2 className="font-display text-lg font-semibold text-text-primary mb-4">
                提现到余额
              </h2>
              
              <div className="mb-4">
                <label className="block text-sm text-text-secondary mb-2">
                  提现金额 (最低 ¥{inviteInfo.min_withdrawal})
                </label>
                <div className="relative">
                  <span className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted">¥</span>
                  <input
                    type="number"
                    value={withdrawAmount}
                    onChange={(e) => setWithdrawAmount(e.target.value)}
                    placeholder="输入金额"
                    min={inviteInfo.min_withdrawal}
                    max={inviteInfo.affiliate_balance}
                    className="w-full pl-8 pr-4 py-3 bg-bg-tertiary border border-white/10 rounded-xl text-text-primary placeholder-text-muted focus:outline-none focus:border-accent/50"
                  />
                </div>
              </div>

              <button
                onClick={handleWithdraw}
                disabled={isWithdrawing || !withdrawAmount || parseFloat(withdrawAmount) < inviteInfo.min_withdrawal || parseFloat(withdrawAmount) > inviteInfo.affiliate_balance}
                className="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-accent hover:bg-accent-hover text-white font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isWithdrawing ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    处理中...
                  </>
                ) : (
                  <>
                    <Wallet className="w-4 h-4" />
                    立即提现
                  </>
                )}
              </button>

              <div className="mt-4 space-y-2 text-xs text-text-tertiary">
                <div className="flex justify-between">
                  <span>最低提现:</span>
                  <span>¥{inviteInfo.min_withdrawal}</span>
                </div>
                <div className="flex justify-between">
                  <span>可提现:</span>
                  <span>¥{inviteInfo.affiliate_balance.toFixed(2)}</span>
                </div>
              </div>
            </div>

            {/* 返利规则 */}
            <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
              <h2 className="font-display text-lg font-semibold text-text-primary mb-4">
                返利规则
              </h2>
              <div className="space-y-3 text-sm">
                <div className="flex items-center justify-between p-3 bg-bg-tertiary/50 rounded-lg">
                  <span className="text-text-secondary">注册奖励</span>
                  <span className="text-accent font-medium">¥{inviteInfo.register_reward}/人</span>
                </div>
                <div className="flex items-center justify-between p-3 bg-bg-tertiary/50 rounded-lg">
                  <span className="text-text-secondary">充值返利</span>
                  <span className="text-accent font-medium">{(inviteInfo.recharge_rate * 100).toFixed(0)}%</span>
                </div>
                <div className="flex items-center justify-between p-3 bg-bg-tertiary/50 rounded-lg">
                  <span className="text-text-secondary">消费返利</span>
                  <span className="text-accent font-medium">{(inviteInfo.consumption_rate * 100).toFixed(0)}%</span>
                </div>
              </div>
            </div>
          </div>

          {/* 右侧：标签页内容 */}
          <div className="lg:col-span-2">
            {/* 标签页切换 */}
            <div className="flex gap-2 mb-6 border-b border-white/5">
              <TabButton
                active={activeTab === 'overview'}
                onClick={() => setActiveTab('overview')}
                label="概览"
              />
              <TabButton
                active={activeTab === 'invites'}
                onClick={() => setActiveTab('invites')}
                label={`邀请记录 (${inviteInfo.invite_count})`}
              />
              <TabButton
                active={activeTab === 'records'}
                onClick={() => setActiveTab('records')}
                label="返利明细"
              />
            </div>

            {/* 概览 */}
            {activeTab === 'overview' && (
              <div className="space-y-6">
                {/* 最近邀请 */}
                <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-display font-semibold text-text-primary">
                      最近邀请
                    </h3>
                    <button
                      onClick={() => setActiveTab('invites')}
                      className="text-sm text-accent hover:text-accent-hover flex items-center gap-1"
                    >
                      查看全部
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                  
                  {inviteRecords.length === 0 ? (
                    <div className="text-center py-8 text-text-tertiary">
                      <Users className="w-12 h-12 mx-auto mb-3 opacity-30" />
                      <p>还没有邀请记录</p>
                      <p className="text-sm mt-1">分享您的邀请链接给好友吧</p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {inviteRecords.slice(0, 5).map((record) => (
                        <div
                          key={record.user_id}
                          className="flex items-center justify-between p-3 bg-bg-tertiary/50 rounded-lg"
                        >
                          <div>
                            <p className="text-text-primary font-medium">
                              {record.user_name || '匿名用户'}
                            </p>
                            <p className="text-sm text-text-tertiary">{record.email}</p>
                          </div>
                          <span className="text-sm text-text-tertiary">
                            {new Date(record.created_at).toLocaleDateString()}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* 最近返利 */}
                <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-display font-semibold text-text-primary">
                      最近返利
                    </h3>
                    <button
                      onClick={() => setActiveTab('records')}
                      className="text-sm text-accent hover:text-accent-hover flex items-center gap-1"
                    >
                      查看全部
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                  
                  {rebateRecords.length === 0 ? (
                    <div className="text-center py-8 text-text-tertiary">
                      <DollarSign className="w-12 h-12 mx-auto mb-3 opacity-30" />
                      <p>还没有返利记录</p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {rebateRecords.slice(0, 5).map((record) => (
                        <div
                          key={record.id}
                          className="flex items-center justify-between p-3 bg-bg-tertiary/50 rounded-lg"
                        >
                          <div>
                            <p className="text-text-primary font-medium">
                              {record.description}
                            </p>
                            <p className="text-sm text-text-tertiary">
                              {new Date(record.created_at).toLocaleDateString()}
                            </p>
                          </div>
                          <span className="text-accent font-medium">
                            +¥{record.amount.toFixed(2)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* 邀请记录 */}
            {activeTab === 'invites' && (
              <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
                <h3 className="font-display font-semibold text-text-primary mb-4">
                  邀请记录
                </h3>
                
                {inviteRecords.length === 0 ? (
                  <div className="text-center py-12 text-text-tertiary">
                    <Users className="w-16 h-16 mx-auto mb-4 opacity-30" />
                    <p className="text-lg">还没有邀请记录</p>
                    <p className="text-sm mt-2">分享您的邀请链接给好友，开始赚取返利</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {inviteRecords.map((record) => (
                      <div
                        key={record.user_id}
                        className="flex items-center justify-between p-4 bg-bg-tertiary/50 rounded-xl"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-accent/10 flex items-center justify-center">
                            <span className="text-accent font-medium">
                              {(record.user_name || 'U')[0].toUpperCase()}
                            </span>
                          </div>
                          <div>
                            <p className="text-text-primary font-medium">
                              {record.user_name || '匿名用户'}
                            </p>
                            <p className="text-sm text-text-tertiary">{record.email}</p>
                          </div>
                        </div>
                        <span className="text-sm text-text-tertiary">
                          {new Date(record.created_at).toLocaleDateString()}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* 返利明细 */}
            {activeTab === 'records' && (
              <div className="bg-bg-secondary border border-white/5 rounded-2xl p-5 sm:p-6">
                <h3 className="font-display font-semibold text-text-primary mb-4">
                  返利明细
                </h3>
                
                {rebateRecords.length === 0 ? (
                  <div className="text-center py-12 text-text-tertiary">
                    <DollarSign className="w-16 h-16 mx-auto mb-4 opacity-30" />
                    <p className="text-lg">还没有返利记录</p>
                    <p className="text-sm mt-2">邀请好友注册和消费，获得返利奖励</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {rebateRecords.map((record) => (
                      <div
                        key={record.id}
                        className="flex items-center justify-between p-4 bg-bg-tertiary/50 rounded-xl"
                      >
                        <div className="flex items-center gap-3">
                          <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                            record.type === 'register' ? 'bg-green-500/10 text-green-500' :
                            record.type === 'recharge' ? 'bg-blue-500/10 text-blue-500' :
                            'bg-accent/10 text-accent'
                          }`}>
                            {record.type === 'register' ? <Gift className="w-5 h-5" /> :
                             record.type === 'recharge' ? <TrendingUp className="w-5 h-5" /> :
                             <DollarSign className="w-5 h-5" />}
                          </div>
                          <div>
                            <p className="text-text-primary font-medium">
                              {record.description}
                            </p>
                            <p className="text-sm text-text-tertiary">
                              {new Date(record.created_at).toLocaleString()}
                            </p>
                          </div>
                        </div>
                        <span className="text-accent font-semibold">
                          +¥{record.amount.toFixed(2)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// 统计卡片组件
function StatCard({ 
  icon, 
  label, 
  value, 
  color 
}: { 
  icon: React.ReactNode; 
  label: string; 
  value: string;
  color: 'accent' | 'accent-2' | 'blue' | 'purple';
}) {
  const colorClasses = {
    'accent': 'bg-accent/10 text-accent',
    'accent-2': 'bg-accent-2/10 text-accent-2',
    'blue': 'bg-blue-500/10 text-blue-500',
    'purple': 'bg-purple-500/10 text-purple-500',
  };

  return (
    <div className="bg-bg-secondary border border-white/5 rounded-xl p-4">
      <div className={`inline-flex items-center justify-center w-10 h-10 rounded-lg mb-3 ${colorClasses[color]}`}>
        {icon}
      </div>
      <p className="text-text-tertiary text-sm">{label}</p>
      <p className="text-xl sm:text-2xl font-display font-bold text-text-primary mt-1">
        {value}
      </p>
    </div>
  );
}

// 标签按钮组件
function TabButton({ 
  active, 
  onClick, 
  label 
}: { 
  active: boolean; 
  onClick: () => void; 
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-3 text-sm font-medium transition-colors relative ${
        active ? 'text-accent' : 'text-text-tertiary hover:text-text-secondary'
      }`}
    >
      {label}
      {active && (
        <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-accent" />
      )}
    </button>
  );
}
