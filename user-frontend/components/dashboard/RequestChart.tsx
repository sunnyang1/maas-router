'use client';

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

interface RequestChartProps {
  data?: Array<{
    date: string;
    requests: number;
    cost?: number;
    tokens?: number;
  }>;
}

export function RequestChart({ data = [] }: RequestChartProps) {
  const formattedData = data.map((item) => ({
    ...item,
    date: new Date(item.date).toLocaleDateString('zh-CN', {
      month: 'short',
      day: 'numeric',
    }),
    // 如果没有 tokens 字段，使用 cost 或 0
    tokens: item.tokens ?? (item.cost ? Math.round(item.cost * 1000) : 0),
  }));

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={formattedData}>
          <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
          <XAxis
            dataKey="date"
            stroke="#64748b"
            fontSize={12}
            tickLine={false}
          />
          <YAxis
            stroke="#64748b"
            fontSize={12}
            tickLine={false}
            axisLine={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: '#1e293b',
              border: '1px solid #334155',
              borderRadius: '8px',
            }}
            labelStyle={{ color: '#94a3b8' }}
          />
          <Legend />
          <Line
            type="monotone"
            dataKey="requests"
            name="请求数"
            stroke="#ff6b35"
            strokeWidth={2}
            dot={{ fill: '#ff6b35', strokeWidth: 0, r: 4 }}
            activeDot={{ r: 6, fill: '#ff6b35' }}
          />
          <Line
            type="monotone"
            dataKey="tokens"
            name="Tokens"
            stroke="#14b8a6"
            strokeWidth={2}
            dot={{ fill: '#14b8a6', strokeWidth: 0, r: 4 }}
            activeDot={{ r: 6, fill: '#14b8a6' }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
