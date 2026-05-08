-- MaaS-Router Database Schema
-- Version: 1.0.0
-- PostgreSQL 16+

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    tier VARCHAR(20) DEFAULT 'free' CHECK (tier IN ('free', 'pro', 'enterprise')),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    cred_balance DECIMAL(18, 6) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tier ON users(tier);
CREATE INDEX idx_users_status ON users(status);

-- API Keys Table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(8) NOT NULL,
    name VARCHAR(100),
    daily_limit DECIMAL(18, 6),
    monthly_limit DECIMAL(18, 6),
    allowed_models TEXT[],
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- Providers Table
CREATE TABLE providers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('self_hosted', 'commercial')),
    endpoint VARCHAR(255) NOT NULL,
    api_key_enc TEXT,
    pricing_model JSONB NOT NULL DEFAULT '{}',
    health_status VARCHAR(20) DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'degraded', 'down', 'unknown')),
    last_check_at TIMESTAMP WITH TIME ZONE,
    weight INTEGER DEFAULT 100,
    max_qps INTEGER DEFAULT 100,
    timeout_ms INTEGER DEFAULT 30000,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_providers_type ON providers(type);
CREATE INDEX idx_providers_status ON providers(status);

-- Request Logs Table (Partitioned)
CREATE TABLE request_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id),
    request_id VARCHAR(64) NOT NULL,
    user_id UUID REFERENCES users(id),
    model VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    complexity_score INTEGER,
    router_reason TEXT,
    router_confidence DECIMAL(3, 2),
    latency_ms INTEGER,
    first_token_ms INTEGER,
    cost DECIMAL(18, 6) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'failed', 'timeout')),
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- Create partitions for current and next month
CREATE TABLE request_logs_y2024m01 PARTITION OF request_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE request_logs_y2024m02 PARTITION OF request_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE INDEX idx_request_logs_api_key_id ON request_logs(api_key_id);
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX idx_request_logs_provider ON request_logs(provider);
CREATE INDEX idx_request_logs_user_id ON request_logs(user_id);

-- Invoices Table
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    subtotal DECIMAL(18, 6) NOT NULL,
    discount DECIMAL(18, 6) DEFAULT 0,
    tax DECIMAL(18, 6) DEFAULT 0,
    total DECIMAL(18, 6) NOT NULL,
    payment_status VARCHAR(20) DEFAULT 'pending' CHECK (payment_status IN ('pending', 'paid', 'failed', 'refunded')),
    payment_method VARCHAR(50),
    paid_at TIMESTAMP WITH TIME ZONE,
    settlement_tx VARCHAR(100),
    merkle_root VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoices_user_id ON invoices(user_id);
CREATE INDEX idx_invoices_period ON invoices(period_start, period_end);

-- Transactions Table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('credit', 'debit', 'refund')),
    amount DECIMAL(18, 6) NOT NULL,
    balance_after DECIMAL(18, 6) NOT NULL,
    description TEXT,
    reference_id VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

-- Router Rules Table
CREATE TABLE router_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    condition JSONB NOT NULL,
    action JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_router_rules_priority ON router_rules(priority);
CREATE INDEX idx_router_rules_active ON router_rules(is_active);

-- System Settings Table
CREATE TABLE system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings
INSERT INTO system_settings (key, value, description) VALUES
('router.default_strategy', '"auto"', 'Default routing strategy'),
('router.fallback_chain', '["self_hosted", "deepseek_api", "azure_openai"]', 'Fallback provider chain'),
('billing.settlement_hour', '0', 'Daily settlement hour (UTC)'),
('billing.min_balance', '1.0', 'Minimum balance threshold');

-- Insert default providers
INSERT INTO providers (id, name, type, endpoint, pricing_model, weight, max_qps) VALUES
('self_hosted_ds_v4', 'DeepSeek-V4 Self-Hosted', 'self_hosted', 'http://vllm-cluster:8000', 
 '{"input": 0.0000005, "output": 0.0000015}', 100, 500),
('deepseek_api', 'DeepSeek API', 'commercial', 'https://api.deepseek.com', 
 '{"input": 0.000001, "output": 0.000002}', 80, 200),
('azure_openai', 'Azure OpenAI', 'commercial', 'https://api.openai.azure.com', 
 '{"input": 0.00001, "output": 0.00003}', 60, 100);

-- Insert default router rules
INSERT INTO router_rules (name, description, priority, condition, action) VALUES
('simple_to_self_hosted', 'Route simple requests to self-hosted cluster', 100,
 '{"complexity_score": {"lte": 4}, "prompt_length": {"lte": 1000}}',
 '{"target": "self_hosted_ds_v4", "fallback": ["deepseek_api"]}'),
('complex_to_premium', 'Route complex requests to premium APIs', 50,
 '{"complexity_score": {"gte": 8}}',
 '{"target": "azure_openai", "fallback": ["deepseek_api"]}');

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_providers_updated_at BEFORE UPDATE ON providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_router_rules_updated_at BEFORE UPDATE ON router_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create view for usage statistics
CREATE VIEW v_usage_stats AS
SELECT 
    user_id,
    DATE(created_at) as date,
    COUNT(*) as request_count,
    SUM(total_tokens) as total_tokens,
    SUM(cost) as total_cost,
    AVG(latency_ms) as avg_latency
FROM request_logs
GROUP BY user_id, DATE(created_at);

-- Create view for provider health summary
CREATE VIEW v_provider_health AS
SELECT 
    provider,
    COUNT(*) as total_requests,
    SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
    AVG(latency_ms) as avg_latency,
    MAX(created_at) as last_request_at
FROM request_logs
WHERE created_at >= NOW() - INTERVAL '1 hour'
GROUP BY provider;