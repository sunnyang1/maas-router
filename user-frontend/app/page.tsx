'use client';

import Link from 'next/link';
import { ArrowRight, Zap, Shield, Coins, Cpu, Sparkles, TrendingDown, Clock, Activity } from 'lucide-react';

export default function Home() {
  return (
    <main className="min-h-screen bg-bg-primary grain">
      {/* Hero Section */}
      <section className="relative min-h-screen flex items-center justify-center overflow-hidden">
        {/* Background Effects */}
        <div className="absolute inset-0 bg-gradient-radial" />
        <div className="absolute inset-0 bg-grid opacity-50" />
        
        {/* Animated gradient orbs */}
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-accent/20 rounded-full blur-[120px] animate-pulse" />
        <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-accent-2/15 rounded-full blur-[100px] animate-pulse" style={{ animationDelay: '1s' }} />
        
        <div className="relative z-10 container-custom">
          <div className="text-center max-w-5xl mx-auto">
            {/* Badge */}
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-bg-tertiary border border-white/10 mb-8 animate-fade-in-up">
              <Sparkles className="w-4 h-4 text-accent" />
              <span className="text-sm text-text-secondary font-medium">v1.0 正式发布</span>
            </div>
            
            {/* Main Title */}
            <h1 className="font-display text-display-1 md:text-[5.5rem] font-bold mb-6 animate-fade-in-up stagger-1">
              <span className="gradient-text">MaaS Router</span>
            </h1>
            
            {/* Subtitle */}
            <p className="text-body-large md:text-2xl text-text-secondary mb-4 max-w-3xl mx-auto animate-fade-in-up stagger-2">
              智能路由降本 · 自建模型托管 · Web3透明对账
            </p>
            
            {/* Description */}
            <p className="text-body text-text-tertiary mb-12 max-w-2xl mx-auto animate-fade-in-up stagger-3">
              通过自研 Judge Agent 智能判断请求复杂度，将 60%+ 简单请求路由至低成本自建 DeepSeek-V4 集群，
              实现整体推理成本降低 40-60%
            </p>
            
            {/* CTA Buttons */}
            <div className="flex flex-col sm:flex-row gap-4 justify-center animate-fade-in-up stagger-4">
              <Link
                href="/register"
                className="group inline-flex items-center justify-center gap-2 px-8 py-4 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold transition-all duration-300 hover:shadow-glow-lg hover:-translate-y-1"
              >
                免费开始使用
                <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
              </Link>
              <Link
                href="/docs"
                className="inline-flex items-center justify-center gap-2 px-8 py-4 rounded-xl bg-bg-tertiary border border-white/10 hover:border-white/20 text-text-primary font-semibold transition-all duration-300 hover:bg-bg-elevated"
              >
                查看文档
              </Link>
            </div>
          </div>
          
          {/* Stats Row */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6 mt-20 animate-fade-in-up stagger-5">
            <StatCard icon={<TrendingDown />} value="40-60%" label="成本降低" />
            <StatCard icon={<Activity />} value="95%+" label="路由准确率" />
            <StatCard icon={<Clock />} value="<30s" label="故障切换" />
            <StatCard icon={<Shield />} value="99.9%+" label="SLA 保障" />
          </div>
        </div>
        
        {/* Scroll indicator */}
        <div className="absolute bottom-8 left-1/2 -translate-x-1/2 animate-bounce">
          <div className="w-6 h-10 rounded-full border-2 border-white/20 flex items-start justify-center p-2">
            <div className="w-1 h-2 bg-accent rounded-full animate-pulse" />
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="section-padding bg-bg-secondary relative">
        <div className="absolute inset-0 bg-grid opacity-30" />
        
        <div className="container-custom relative z-10">
          <div className="text-center mb-16">
            <span className="text-accent text-sm font-semibold uppercase tracking-wider mb-4 block">核心特性</span>
            <h2 className="font-display text-display-3 mb-4">全方位的 AI 推理优化解决方案</h2>
            <p className="text-text-secondary max-w-2xl mx-auto">集成智能路由、自建集群和区块链结算，打造企业级 AI API 网关</p>
          </div>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            <FeatureCard
              icon={<Zap className="w-7 h-7" />}
              title="OpenAI 兼容 API"
              description="零代码迁移，只需修改 base_url 即可接入，支持所有 OpenAI SDK"
              delay={0}
            />
            <FeatureCard
              icon={<Cpu className="w-7 h-7" />}
              title="智能路由 Agent"
              description="基于 Qwen2.5-7B 的复杂度评分系统，路由准确率目标 95%+"
              delay={0.1}
            />
            <FeatureCard
              icon={<Shield className="w-7 h-7" />}
              title="自建推理集群"
              description="DeepSeek-V4 集群，成本低于商业 API 40%+"
              delay={0.2}
            />
            <FeatureCard
              icon={<Coins className="w-7 h-7" />}
              title="$CRED 代币体系"
              description="链下实时计费 + L2 每日结算，透明可验证"
              delay={0.3}
            />
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section className="section-padding relative">
        <div className="absolute inset-0 bg-gradient-radial opacity-50" />
        
        <div className="container-custom relative z-10">
          <div className="text-center mb-16">
            <span className="text-accent text-sm font-semibold uppercase tracking-wider mb-4 block">模型定价</span>
            <h2 className="font-display text-display-3 mb-4">透明定价，按量付费</h2>
            <p className="text-text-secondary">智能路由自动选择最优模型，平衡成本与性能</p>
          </div>
          
          <div className="grid md:grid-cols-3 gap-6 max-w-5xl mx-auto">
            <PricingCard
              model="deepseek-v4"
              name="DeepSeek-V4"
              type="自建推理"
              price="$0.5"
              description="高性价比自建集群"
              features={["智能路由优选", "低延迟响应", "成本最优"]}
              highlighted
            />
            <PricingCard
              model="deepseek-api"
              name="DeepSeek API"
              type="商业 API"
              price="$1.0"
              description="官方 API 服务"
              features={["稳定可靠", "完整功能", "专业支持"]}
            />
            <PricingCard
              model="gpt-4-turbo"
              name="GPT-4 Turbo"
              type="商业 API"
              price="$10.0"
              description="OpenAI 旗舰模型"
              features={["最强性能", "复杂推理", "高级创作"]}
            />
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="section-padding bg-bg-secondary relative overflow-hidden">
        <div className="absolute inset-0 bg-grid opacity-30" />
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[600px] h-[300px] bg-accent/10 rounded-full blur-[100px]" />
        
        <div className="container-custom relative z-10 text-center">
          <h2 className="font-display text-display-2 mb-6">准备好开始了吗？</h2>
          <p className="text-text-secondary text-body-large mb-10 max-w-2xl mx-auto">
            立即注册，获得 $10 CRED 免费额度，体验智能路由带来的成本优化
          </p>
          <Link
            href="/register"
            className="group inline-flex items-center justify-center gap-2 px-10 py-5 rounded-xl bg-accent hover:bg-accent-hover text-white font-semibold text-lg transition-all duration-300 hover:shadow-glow-lg hover:-translate-y-1"
          >
            免费注册
            <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 border-t border-white/5 bg-bg-primary">
        <div className="container-custom">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="font-display text-xl font-bold gradient-text">MaaS Router</span>
            </div>
            <p className="text-text-muted text-sm">
              © 2026 MaaS Router. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </main>
  );
}

function StatCard({ icon, value, label }: { icon: React.ReactNode; value: string; label: string }) {
  return (
    <div className="text-center p-6 rounded-2xl bg-bg-tertiary/50 border border-white/5">
      <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-accent/10 text-accent mb-4">
        {icon}
      </div>
      <div className="font-display text-3xl font-bold gradient-text mb-1">{value}</div>
      <div className="text-text-tertiary text-sm">{label}</div>
    </div>
  );
}

function FeatureCard({ 
  icon, 
  title, 
  description, 
  delay 
}: { 
  icon: React.ReactNode; 
  title: string; 
  description: string;
  delay: number;
}) {
  return (
    <div 
      className="group p-6 rounded-2xl bg-bg-tertiary border border-white/5 card-hover"
      style={{ animationDelay: `${delay}s` }}
    >
      <div className="inline-flex items-center justify-center w-14 h-14 rounded-xl bg-accent/10 text-accent mb-5 group-hover:scale-110 transition-transform duration-300">
        {icon}
      </div>
      <h3 className="font-display text-heading-3 mb-3 text-text-primary">{title}</h3>
      <p className="text-text-secondary text-body-small leading-relaxed">{description}</p>
    </div>
  );
}

function PricingCard({
  model,
  name,
  type,
  price,
  description,
  features,
  highlighted = false,
}: {
  model: string;
  name: string;
  type: string;
  price: string;
  description: string;
  features: string[];
  highlighted?: boolean;
}) {
  return (
    <div
      className={`relative p-8 rounded-2xl border transition-all duration-300 ${
        highlighted
          ? 'bg-gradient-to-b from-accent/10 to-transparent border-accent/30 shadow-glow'
          : 'bg-bg-tertiary border-white/5 hover:border-white/10'
      }`}
    >
      {highlighted && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <span className="px-4 py-1 rounded-full bg-accent text-white text-xs font-semibold">
            推荐
          </span>
        </div>
      )}
      
      <div className="mb-6">
        <span className="inline-block px-3 py-1 rounded-full bg-bg-elevated text-text-tertiary text-xs font-medium mb-4">
          {type}
        </span>
        <h3 className="font-display text-heading-1 mb-2">{name}</h3>
        <p className="text-text-secondary text-body-small">{description}</p>
      </div>
      
      <div className="mb-6">
        <span className="font-display text-4xl font-bold text-text-primary">{price}</span>
        <span className="text-text-tertiary"> / 1M tokens</span>
      </div>
      
      <ul className="space-y-3 mb-8">
        {features.map((feature, i) => (
          <li key={i} className="flex items-center gap-3 text-body-small text-text-secondary">
            <span className="w-1.5 h-1.5 rounded-full bg-accent flex-shrink-0" />
            {feature}
          </li>
        ))}
      </ul>
      
      <Link
        href={`/dashboard?model=${model}`}
        className={`block text-center py-3 rounded-xl font-semibold transition-all duration-300 ${
          highlighted
            ? 'bg-accent hover:bg-accent-hover text-white hover:shadow-glow'
            : 'bg-bg-elevated hover:bg-bg-secondary text-text-primary border border-white/10'
        }`}
      >
        开始使用
      </Link>
    </div>
  );
}
