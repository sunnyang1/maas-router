/**
 * 管理后台仪表盘页面
 * 包含多种统计图表、实时数据刷新功能
 * 使用 ECharts 进行数据可视化
 */
import React, { useEffect, useRef, useState } from 'react';
import { Card, Row, Col, Statistic, DatePicker, Select, Space, Badge, Tooltip, Spin } from 'antd';
import {
  LineChartOutlined,
  BarChartOutlined,
  PieChartOutlined,
  RiseOutlined,
  FallOutlined,
  UserOutlined,
  ApiOutlined,
  DollarOutlined,
  WarningOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import * as echarts from 'echarts';
import type { ECharts } from 'echarts';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;
const { Option } = Select;

/**
 * 统计数据类型
 */
interface StatisticsData {
  totalRequests: number;
  totalUsers: number;
  totalRevenue: number;
  errorRate: number;
  requestsGrowth: number;
  usersGrowth: number;
  revenueGrowth: number;
  errorRateChange: number;
}

/**
 * 图表数据类型
 */
interface ChartData {
  dates: string[];
  requests: number[];
  users: number[];
  revenue: number[];
  errors: number[];
  routerDistribution: { name: string; value: number }[];
  modelUsage: { name: string; value: number }[];
}

/**
 * 仪表盘页面组件
 */
const Dashboard: React.FC = () => {
  // 图表容器引用
  const requestChartRef = useRef<HTMLDivElement>(null);
  const userChartRef = useRef<HTMLDivElement>(null);
  const revenueChartRef = useRef<HTMLDivElement>(null);
  const routerPieChartRef = useRef<HTMLDivElement>(null);
  const modelBarChartRef = useRef<HTMLDivElement>(null);
  const errorChartRef = useRef<HTMLDivElement>(null);

  // ECharts 实例引用
  const chartsRef = useRef<{ [key: string]: ECharts | null }>({});

  // 状态管理
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(7, 'day'),
    dayjs(),
  ]);
  const [refreshInterval, setRefreshInterval] = useState<number>(30);
  const [statistics, setStatistics] = useState<StatisticsData>({
    totalRequests: 1234567,
    totalUsers: 45678,
    totalRevenue: 98765.43,
    errorRate: 0.23,
    requestsGrowth: 12.5,
    usersGrowth: 8.3,
    revenueGrowth: 15.2,
    errorRateChange: -0.05,
  });

  // 模拟图表数据
  const [chartData, setChartData] = useState<ChartData>({
    dates: [],
    requests: [],
    users: [],
    revenue: [],
    errors: [],
    routerDistribution: [
      { name: 'GPT-4', value: 4500 },
      { name: 'GPT-3.5', value: 3200 },
      { name: 'Claude', value: 2800 },
      { name: 'Gemini', value: 1500 },
      { name: '其他', value: 1000 },
    ],
    modelUsage: [
      { name: '文本生成', value: 5234 },
      { name: '代码补全', value: 3456 },
      { name: '图像生成', value: 2134 },
      { name: '语音识别', value: 1567 },
      { name: '翻译', value: 1234 },
    ],
  });

  /**
   * 生成模拟数据
   */
  const generateMockData = (): ChartData => {
    const dates: string[] = [];
    const requests: number[] = [];
    const users: number[] = [];
    const revenue: number[] = [];
    const errors: number[] = [];

    const days = timeRange[1].diff(timeRange[0], 'day');
    
    for (let i = 0; i <= days; i++) {
      const date = timeRange[0].add(i, 'day').format('MM-DD');
      dates.push(date);
      requests.push(Math.floor(Math.random() * 5000) + 3000);
      users.push(Math.floor(Math.random() * 500) + 200);
      revenue.push(Math.floor(Math.random() * 10000) + 5000);
      errors.push(Math.floor(Math.random() * 50));
    }

    return {
      ...chartData,
      dates,
      requests,
      users,
      revenue,
      errors,
    };
  };

  /**
   * 初始化图表
   */
  const initCharts = () => {
    if (requestChartRef.current) {
      chartsRef.current.request = echarts.init(requestChartRef.current);
    }
    if (userChartRef.current) {
      chartsRef.current.user = echarts.init(userChartRef.current);
    }
    if (revenueChartRef.current) {
      chartsRef.current.revenue = echarts.init(revenueChartRef.current);
    }
    if (routerPieChartRef.current) {
      chartsRef.current.routerPie = echarts.init(routerPieChartRef.current);
    }
    if (modelBarChartRef.current) {
      chartsRef.current.modelBar = echarts.init(modelBarChartRef.current);
    }
    if (errorChartRef.current) {
      chartsRef.current.error = echarts.init(errorChartRef.current);
    }
  };

  /**
   * 更新请求趋势图表
   */
  const updateRequestChart = () => {
    const chart = chartsRef.current.request;
    if (!chart) return;

    const option: echarts.EChartsOption = {
      title: {
        text: 'API 请求趋势',
        left: 'center',
        textStyle: { fontSize: 14 },
      },
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
      },
      legend: {
        data: ['请求量', '用户数'],
        bottom: 0,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '15%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: chartData.dates,
      },
      yAxis: [
        {
          type: 'value',
          name: '请求量',
          position: 'left',
        },
        {
          type: 'value',
          name: '用户数',
          position: 'right',
        },
      ],
      series: [
        {
          name: '请求量',
          type: 'line',
          smooth: true,
          areaStyle: {
            opacity: 0.3,
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#1890ff' },
              { offset: 1, color: '#1890ff20' },
            ]),
          },
          data: chartData.requests,
          itemStyle: { color: '#1890ff' },
        },
        {
          name: '用户数',
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: chartData.users,
          itemStyle: { color: '#52c41a' },
        },
      ],
    };

    chart.setOption(option);
  };

  /**
   * 更新收入图表
   */
  const updateRevenueChart = () => {
    const chart = chartsRef.current.revenue;
    if (!chart) return;

    const option: echarts.EChartsOption = {
      title: {
        text: '收入统计',
        left: 'center',
        textStyle: { fontSize: 14 },
      },
      tooltip: {
        trigger: 'axis',
        formatter: '{b}<br/>{a}: ¥{c}',
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: chartData.dates,
      },
      yAxis: {
        type: 'value',
        axisLabel: {
          formatter: '¥{value}',
        },
      },
      series: [
        {
          name: '收入',
          type: 'bar',
          data: chartData.revenue,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#faad14' },
              { offset: 1, color: '#faad1460' },
            ]),
            borderRadius: [4, 4, 0, 0],
          },
        },
      ],
    };

    chart.setOption(option);
  };

  /**
   * 更新路由分布饼图
   */
  const updateRouterPieChart = () => {
    const chart = chartsRef.current.routerPie;
    if (!chart) return;

    const option: echarts.EChartsOption = {
      title: {
        text: '模型路由分布',
        left: 'center',
        textStyle: { fontSize: 14 },
      },
      tooltip: {
        trigger: 'item',
        formatter: '{a} <br/>{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'vertical',
        left: 'left',
        top: 'middle',
      },
      series: [
        {
          name: '模型使用',
          type: 'pie',
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 10,
            borderColor: '#fff',
            borderWidth: 2,
          },
          label: {
            show: false,
            position: 'center',
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold',
            },
          },
          labelLine: { show: false },
          data: chartData.routerDistribution,
        },
      ],
    };

    chart.setOption(option);
  };

  /**
   * 更新模型使用柱状图
   */
  const updateModelBarChart = () => {
    const chart = chartsRef.current.modelBar;
    if (!chart) return;

    const option: echarts.EChartsOption = {
      title: {
        text: '功能使用统计',
        left: 'center',
        textStyle: { fontSize: 14 },
      },
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true,
      },
      xAxis: {
        type: 'value',
      },
      yAxis: {
        type: 'category',
        data: chartData.modelUsage.map(item => item.name).reverse(),
      },
      series: [
        {
          name: '使用次数',
          type: 'bar',
          data: chartData.modelUsage.map(item => item.value).reverse(),
          itemStyle: {
            color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
              { offset: 0, color: '#722ed1' },
              { offset: 1, color: '#722ed160' },
            ]),
            borderRadius: [0, 4, 4, 0],
          },
          label: {
            show: true,
            position: 'right',
          },
        },
      ],
    };

    chart.setOption(option);
  };

  /**
   * 更新错误率图表
   */
  const updateErrorChart = () => {
    const chart = chartsRef.current.error;
    if (!chart) return;

    const option: echarts.EChartsOption = {
      title: {
        text: '错误率趋势',
        left: 'center',
        textStyle: { fontSize: 14 },
      },
      tooltip: {
        trigger: 'axis',
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '10%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: chartData.dates,
      },
      yAxis: {
        type: 'value',
        axisLabel: {
          formatter: '{value}%',
        },
      },
      visualMap: {
        show: false,
        pieces: [
          { gt: 0, lte: 5, color: '#52c41a' },
          { gt: 5, lte: 10, color: '#faad14' },
          { gt: 10, color: '#f5222d' },
        ],
      },
      series: [
        {
          name: '错误率',
          type: 'line',
          smooth: true,
          data: chartData.errors.map(v => (v / 100).toFixed(2)),
          areaStyle: { opacity: 0.3 },
        },
      ],
    };

    chart.setOption(option);
  };

  /**
   * 刷新数据
   */
  const refreshData = async () => {
    setLoading(true);
    // 模拟 API 请求
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    const newData = generateMockData();
    setChartData(newData);
    
    // 更新统计数据
    setStatistics(prev => ({
      ...prev,
      totalRequests: prev.totalRequests + Math.floor(Math.random() * 1000),
      totalUsers: prev.totalUsers + Math.floor(Math.random() * 50),
    }));
    
    setLoading(false);
  };

  /**
   * 处理窗口大小变化
   */
  const handleResize = () => {
    Object.values(chartsRef.current).forEach(chart => {
      chart?.resize();
    });
  };

  // 初始化
  useEffect(() => {
    initCharts();
    setChartData(generateMockData());

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      Object.values(chartsRef.current).forEach(chart => {
        chart?.dispose();
      });
    };
  }, []);

  // 更新图表
  useEffect(() => {
    updateRequestChart();
    updateRevenueChart();
    updateRouterPieChart();
    updateModelBarChart();
    updateErrorChart();
  }, [chartData]);

  // 自动刷新
  useEffect(() => {
    if (refreshInterval <= 0) return;

    const timer = setInterval(() => {
      refreshData();
    }, refreshInterval * 1000);

    return () => clearInterval(timer);
  }, [refreshInterval, timeRange]);

  return (
    <div style={{ padding: '24px' }}>
      {/* 页面标题和工具栏 */}
      <Row justify="space-between" align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <h1 style={{ margin: 0, fontSize: 24, fontWeight: 500 }}>
            <BarChartOutlined style={{ marginRight: 8 }} />
            数据仪表盘
          </h1>
        </Col>
        <Col>
          <Space>
            <RangePicker
              value={timeRange}
              onChange={dates => {
                if (dates) {
                  setTimeRange([dates[0]!, dates[1]!]);
                  refreshData();
                }
              }}
            />
            <Select
              value={refreshInterval}
              onChange={setRefreshInterval}
              style={{ width: 120 }}
            >
              <Option value={0}>不刷新</Option>
              <Option value={10}>10秒</Option>
              <Option value={30}>30秒</Option>
              <Option value={60}>1分钟</Option>
              <Option value={300}>5分钟</Option>
            </Select>
            <Tooltip title="手动刷新">
              <Badge dot={loading}>
                <ReloadOutlined
                  onClick={refreshData}
                  style={{ fontSize: 18, cursor: 'pointer' }}
                  spin={loading}
                />
              </Badge>
            </Tooltip>
          </Space>
        </Col>
      </Row>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title={
                <Space>
                  <ApiOutlined />
                  总请求量
                </Space>
              }
              value={statistics.totalRequests}
              precision={0}
              valueStyle={{ color: '#1890ff' }}
              suffix={
                <span style={{ fontSize: 14 }}>
                  {statistics.requestsGrowth > 0 ? (
                    <RiseOutlined style={{ color: '#52c41a' }} />
                  ) : (
                    <FallOutlined style={{ color: '#f5222d' }} />
                  )}
                  {Math.abs(statistics.requestsGrowth)}%
                </span>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title={
                <Space>
                  <UserOutlined />
                  总用户数
                </Space>
              }
              value={statistics.totalUsers}
              precision={0}
              valueStyle={{ color: '#52c41a' }}
              suffix={
                <span style={{ fontSize: 14 }}>
                  {statistics.usersGrowth > 0 ? (
                    <RiseOutlined style={{ color: '#52c41a' }} />
                  ) : (
                    <FallOutlined style={{ color: '#f5222d' }} />
                  )}
                  {Math.abs(statistics.usersGrowth)}%
                </span>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title={
                <Space>
                  <DollarOutlined />
                  总收入
                </Space>
              }
              value={statistics.totalRevenue}
              precision={2}
              prefix="¥"
              valueStyle={{ color: '#faad14' }}
              suffix={
                <span style={{ fontSize: 14 }}>
                  {statistics.revenueGrowth > 0 ? (
                    <RiseOutlined style={{ color: '#52c41a' }} />
                  ) : (
                    <FallOutlined style={{ color: '#f5222d' }} />
                  )}
                  {Math.abs(statistics.revenueGrowth)}%
                </span>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title={
                <Space>
                  <WarningOutlined />
                  错误率
                </Space>
              }
              value={statistics.errorRate}
              precision={2}
              suffix="%"
              valueStyle={{
                color: statistics.errorRate < 1 ? '#52c41a' : statistics.errorRate < 5 ? '#faad14' : '#f5222d',
              }}
            />
          </Card>
        </Col>
      </Row>

      {/* 图表区域 */}
      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          {/* 请求趋势图 */}
          <Col xs={24} lg={12}>
            <Card
              title={
                <Space>
                  <LineChartOutlined />
                  请求趋势
                </Space>
              }
            >
              <div ref={requestChartRef} style={{ height: 300 }} />
            </Card>
          </Col>

          {/* 收入统计图 */}
          <Col xs={24} lg={12}>
            <Card
              title={
                <Space>
                  <BarChartOutlined />
                  收入统计
                </Space>
              }
            >
              <div ref={revenueChartRef} style={{ height: 300 }} />
            </Card>
          </Col>

          {/* 路由分布饼图 */}
          <Col xs={24} lg={8}>
            <Card
              title={
                <Space>
                  <PieChartOutlined />
                  模型路由分布
                </Space>
              }
            >
              <div ref={routerPieChartRef} style={{ height: 300 }} />
            </Card>
          </Col>

          {/* 功能使用统计 */}
          <Col xs={24} lg={8}>
            <Card
              title={
                <Space>
                  <BarChartOutlined />
                  功能使用统计
                </Space>
              }
            >
              <div ref={modelBarChartRef} style={{ height: 300 }} />
            </Card>
          </Col>

          {/* 错误率趋势 */}
          <Col xs={24} lg={8}>
            <Card
              title={
                <Space>
                  <LineChartOutlined />
                  错误率趋势
                </Space>
              }
            >
              <div ref={errorChartRef} style={{ height: 300 }} />
            </Card>
          </Col>
        </Row>
      </Spin>

      {/* 实时状态指示器 */}
      <div style={{ marginTop: 16, textAlign: 'center' }}>
        <Space>
          <Badge status="processing" text="数据实时更新中" />
          <span style={{ color: '#999' }}>
            上次更新: {dayjs().format('HH:mm:ss')}
          </span>
        </Space>
      </div>
    </div>
  );
};

export default Dashboard;
