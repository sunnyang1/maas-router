import { TrendingUp, TrendingDown, DollarSign } from 'lucide-react';

interface CostCardProps {
  title: string;
  amount: number;
  currency: string;
  trend?: number; // 改为简单的百分比数值
}

export function CostCard({ title, amount, currency, trend }: CostCardProps) {
  // 将简单的数值转换为趋势对象
  const trendData = trend !== undefined ? {
    value: Math.abs(trend),
    isPositive: trend >= 0,
  } : undefined;

  return (
    <div className="bg-bg-tertiary rounded-2xl border border-white/5 p-6 card-hover">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-body-small text-text-tertiary mb-1">{title}</p>
          <div className="flex items-baseline gap-1">
            <span className="font-display text-2xl font-bold text-text-primary">{amount.toFixed(2)}</span>
            <span className="text-sm text-text-tertiary">{currency}</span>
          </div>
        </div>
        <div className="p-3 rounded-xl bg-accent/10">
          <DollarSign className="w-6 h-6 text-accent" />
        </div>
      </div>
      
      {trendData && (
        <div className="mt-4 flex items-center gap-2">
          {trendData.isPositive ? (
            <TrendingUp className="w-4 h-4 text-accent-2" />
          ) : (
            <TrendingDown className="w-4 h-4 text-accent" />
          )}
          <span className={`text-sm font-medium ${trendData.isPositive ? 'text-accent-2' : 'text-accent'}`}>
            {trendData.isPositive ? '+' : ''}{trendData.value.toFixed(1)}%
          </span>
          <span className="text-sm text-text-muted">较上期</span>
        </div>
      )}
    </div>
  );
}
