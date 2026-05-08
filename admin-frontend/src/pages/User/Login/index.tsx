import React, { useState } from 'react';
import { LoginForm, ProFormText } from '@ant-design/pro-components';
import { message, Tabs } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { history, useModel } from '@umijs/max';
import { login } from '@/services/api';

const Login: React.FC = () => {
  const [type, setType] = useState<string>('account');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      const response = await login(values);
      if (response.success) {
        localStorage.setItem('token', response.data.token);
        message.success('登录成功');
        history.push('/dashboard');
      } else {
        message.error(response.message || '登录失败');
      }
    } catch (error) {
      message.error('登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        height: '100vh',
        background: '#f0f2f5',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <LoginForm
        title="MaaS-Router Admin"
        subTitle="Model as a Service Router 管理系统"
        onFinish={handleSubmit}
        loading={loading}
      >
        <Tabs
          activeKey={type}
          onChange={setType}
          centered
          items={[
            {
              key: 'account',
              label: '账户密码登录',
            },
          ]}
        />

        {type === 'account' && (
          <>
            <ProFormText
              name="username"
              fieldProps={{
                size: 'large',
                prefix: <UserOutlined />,
              }}
              placeholder="用户名"
              rules={[
                {
                  required: true,
                  message: '请输入用户名',
                },
              ]}
            />
            <ProFormText.Password
              name="password"
              fieldProps={{
                size: 'large',
                prefix: <LockOutlined />,
              }}
              placeholder="密码"
              rules={[
                {
                  required: true,
                  message: '请输入密码',
                },
              ]}
            />
          </>
        )}

        <div
          style={{
            marginBottom: 24,
          }}
        >
          <a
            style={{
              float: 'right',
            }}
            onClick={() => message.info('请联系管理员重置密码')}
          >
            忘记密码
          </a>
        </div>
      </LoginForm>
    </div>
  );
};

export default Login;
