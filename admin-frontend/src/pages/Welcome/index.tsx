import React from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Card, Typography, Alert } from 'antd';

const { Title, Paragraph, Text } = Typography;

const Welcome: React.FC = () => {
  return (
    <PageContainer>
      <Card>
        <Alert
          message="欢迎使用 MaaS-Router Admin"
          type="success"
          showIcon
          banner
          style={{
            margin: -12,
            marginBottom: 24,
          }}
        />
        <Typography>
          <Title level={3}>MaaS-Router 管理系统</Title>
          <Paragraph>
            MaaS-Router 是一个 Model as a Service 路由管理系统，提供统一的 API 接口管理多个 LLM 供应商。
          </Paragraph>
          <Paragraph>
            <Text strong>主要功能：</Text>
          </Paragraph>
          <ul>
            <li>用户管理 - 管理系统用户，配置配额和权限</li>
            <li>API Key 管理 - 创建和管理 API 访问密钥</li>
            <li>供应商管理 - 配置 LLM 供应商，支持 OpenAI、Anthropic、Azure 等</li>
            <li>模型管理 - 管理支持的模型，配置定价策略</li>
            <li>计费管理 - 查看账单记录，处理用户充值</li>
            <li>路由规则 - 配置智能路由策略，实现负载均衡</li>
            <li>监控告警 - 实时监控系统状态，配置告警规则</li>
            <li>系统设置 - 配置系统参数，查看运行状态</li>
          </ul>
        </Typography>
      </Card>
    </PageContainer>
  );
};

export default Welcome;
