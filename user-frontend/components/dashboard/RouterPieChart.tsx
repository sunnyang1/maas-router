'use client';

import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';

interface RouterPieChartProps {
  data?: Array<{
    name: string;
    value: number;
  }>;
}

const COLORS = ['#0ea5e9', '#10b981', '#f59e0b', '#ef4444'];

export function RouterPieChart({ data = [] }: RouterPieChartProps) {
  const defaultData = [
    { name: '自建 DeepSeek-V4', value: 65 },
    { name: 'DeepSeek API', value: 25 },
    { name: 'GPT-4 Turbo', value: 8 },
    { name: 'Claude 3', value: 2 },
  ];

  const chartData = data.length > 0 ? data : defaultData;

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={80}
            paddingAngle={5}
            dataKey="value"
          >
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{
              backgroundColor: '#1e293b',
              border: '1px solid #334155',
              borderRadius: '8px',
            }}
            formatter={(value: number) => [`${value}%`, '占比']}
          />
          <Legend
            verticalAlign="bottom"
            height={36}
            iconType="circle"
            wrapperStyle={{ fontSize: 12 }}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}