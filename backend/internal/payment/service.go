// Package payment 提供支付功能的统一接口
// 支持 Stripe、支付宝、微信支付等多种支付方式
package payment

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// PaymentStatus 支付状态枚举
type PaymentStatus string

const (
	// PaymentStatusPending 待支付
	PaymentStatusPending PaymentStatus = "pending"
	// PaymentStatusProcessing 处理中
	PaymentStatusProcessing PaymentStatus = "processing"
	// PaymentStatusSuccess 支付成功
	PaymentStatusSuccess PaymentStatus = "success"
	// PaymentStatusFailed 支付失败
	PaymentStatusFailed PaymentStatus = "failed"
	// PaymentStatusCancelled 已取消
	PaymentStatusCancelled PaymentStatus = "cancelled"
	// PaymentStatusRefunded 已退款
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// PaymentProvider 支付提供商枚举
type PaymentProvider string

const (
	// ProviderStripe Stripe 支付
	ProviderStripe PaymentProvider = "stripe"
	// ProviderAlipay 支付宝
	ProviderAlipay PaymentProvider = "alipay"
	// ProviderWechat 微信支付
	ProviderWechat PaymentProvider = "wechat"
)

// CreatePaymentRequest 创建支付请求
type CreatePaymentRequest struct {
	// 订单ID
	OrderID string `json:"order_id"`
	// 用户ID
	UserID string `json:"user_id"`
	// 支付金额（单位：分）
	Amount int64 `json:"amount"`
	// 货币代码 (CNY, USD, EUR 等)
	Currency string `json:"currency"`
	// 支付提供商
	Provider PaymentProvider `json:"provider"`
	// 商品描述
	Description string `json:"description"`
	// 回调URL
	NotifyURL string `json:"notify_url"`
	// 返回URL（支付完成后跳转）
	ReturnURL string `json:"return_url"`
	// 附加数据
	Metadata map[string]string `json:"metadata"`
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
	// 支付ID
	PaymentID string `json:"payment_id"`
	// 支付状态
	Status PaymentStatus `json:"status"`
	// 支付URL（用于跳转支付页面）
	PaymentURL string `json:"payment_url,omitempty"`
	// 支付参数（用于APP或JSAPI支付）
	PaymentParams map[string]interface{} `json:"payment_params,omitempty"`
	// 过期时间
	ExpireAt time.Time `json:"expire_at"`
}

// QueryPaymentResponse 查询支付状态响应
type QueryPaymentResponse struct {
	// 支付ID
	PaymentID string `json:"payment_id"`
	// 订单ID
	OrderID string `json:"order_id"`
	// 支付状态
	Status PaymentStatus `json:"status"`
	// 支付金额
	Amount int64 `json:"amount"`
	// 货币代码
	Currency string `json:"currency"`
	// 支付提供商
	Provider PaymentProvider `json:"provider"`
	// 第三方支付单号
	ThirdPartyID string `json:"third_party_id,omitempty"`
	// 支付时间
	PaidAt *time.Time `json:"paid_at,omitempty"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// RefundRequest 退款请求
type RefundRequest struct {
	// 支付ID
	PaymentID string `json:"payment_id"`
	// 退款金额（单位：分），0表示全额退款
	Amount int64 `json:"amount"`
	// 退款原因
	Reason string `json:"reason"`
	// 退款单号（用于幂等）
	RefundID string `json:"refund_id"`
}

// RefundResponse 退款响应
type RefundResponse struct {
	// 退款ID
	RefundID string `json:"refund_id"`
	// 退款状态
	Status string `json:"status"`
	// 退款金额
	Amount int64 `json:"amount"`
	// 退款时间
	RefundedAt *time.Time `json:"refunded_at,omitempty"`
}

// WebhookPayload 支付回调数据
type WebhookPayload struct {
	// 支付提供商
	Provider PaymentProvider `json:"provider"`
	// 事件类型
	EventType string `json:"event_type"`
	// 事件数据
	Data map[string]interface{} `json:"data"`
	// 签名（用于验证）
	Signature string `json:"signature"`
}

// Provider 支付提供商接口
type Provider interface {
	// GetName 获取支付提供商名称
	GetName() PaymentProvider
	
	// CreatePayment 创建支付
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	
	// QueryPayment 查询支付状态
	QueryPayment(ctx context.Context, paymentID string) (*QueryPaymentResponse, error)
	
	// CancelPayment 取消支付
	CancelPayment(ctx context.Context, paymentID string) error
	
	// Refund 退款
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
	
	// VerifyWebhook 验证回调签名
	VerifyWebhook(ctx context.Context, payload *WebhookPayload) (bool, error)
	
	// ParseWebhook 解析回调数据
	ParseWebhook(ctx context.Context, body []byte, signature string) (*WebhookPayload, error)
}

// Service 支付服务
type Service struct {
	// 支付提供商映射
	providers map[PaymentProvider]Provider
}

// NewService 创建支付服务实例
func NewService() *Service {
	return &Service{
		providers: make(map[PaymentProvider]Provider),
	}
}

// RegisterProvider 注册支付提供商
func (s *Service) RegisterProvider(provider Provider) {
	s.providers[provider.GetName()] = provider
}

// GetProvider 获取支付提供商
func (s *Service) GetProvider(name PaymentProvider) (Provider, error) {
	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("不支持的支付提供商: %s", name)
	}
	return provider, nil
}

// CreatePayment 创建支付订单
func (s *Service) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 参数校验
	if req.OrderID == "" {
		return nil, errors.New("订单ID不能为空")
	}
	if req.UserID == "" {
		return nil, errors.New("用户ID不能为空")
	}
	if req.Amount <= 0 {
		return nil, errors.New("支付金额必须大于0")
	}
	if req.Currency == "" {
		req.Currency = "CNY"
	}
	
	// 获取支付提供商
	provider, err := s.GetProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	
	// 调用提供商创建支付
	return provider.CreatePayment(ctx, req)
}

// QueryPayment 查询支付状态
func (s *Service) QueryPayment(ctx context.Context, provider PaymentProvider, paymentID string) (*QueryPaymentResponse, error) {
	if paymentID == "" {
		return nil, errors.New("支付ID不能为空")
	}
	
	p, err := s.GetProvider(provider)
	if err != nil {
		return nil, err
	}
	
	return p.QueryPayment(ctx, paymentID)
}

// CancelPayment 取消支付
func (s *Service) CancelPayment(ctx context.Context, provider PaymentProvider, paymentID string) error {
	if paymentID == "" {
		return errors.New("支付ID不能为空")
	}
	
	p, err := s.GetProvider(provider)
	if err != nil {
		return err
	}
	
	return p.CancelPayment(ctx, paymentID)
}

// Refund 退款
func (s *Service) Refund(ctx context.Context, provider PaymentProvider, req *RefundRequest) (*RefundResponse, error) {
	if req.PaymentID == "" {
		return nil, errors.New("支付ID不能为空")
	}
	
	p, err := s.GetProvider(provider)
	if err != nil {
		return nil, err
	}
	
	return p.Refund(ctx, req)
}

// HandleWebhook 处理支付回调
func (s *Service) HandleWebhook(ctx context.Context, provider PaymentProvider, body []byte, signature string) (*WebhookPayload, error) {
	p, err := s.GetProvider(provider)
	if err != nil {
		return nil, err
	}
	
	// 解析回调数据
	payload, err := p.ParseWebhook(ctx, body, signature)
	if err != nil {
		return nil, fmt.Errorf("解析回调数据失败: %w", err)
	}
	
	// 验证签名
	valid, err := p.VerifyWebhook(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("验证回调签名失败: %w", err)
	}
	if !valid {
		return nil, errors.New("回调签名验证失败")
	}
	
	return payload, nil
}

// GetSupportedProviders 获取支持的支付提供商列表
func (s *Service) GetSupportedProviders() []PaymentProvider {
	providers := make([]PaymentProvider, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}
