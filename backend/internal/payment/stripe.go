// Package payment 提供 Stripe 支付集成
package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeConfig Stripe 配置
type StripeConfig struct {
	// SecretKey Stripe 密钥
	SecretKey string
	// WebhookSecret Webhook 签名密钥
	WebhookSecret string
	// APIBase API 基础地址（可选，用于测试）
	APIBase string
}

// StripeProvider Stripe 支付提供商
type StripeProvider struct {
	config *StripeConfig
}

// NewStripeProvider 创建 Stripe 支付提供商
func NewStripeProvider(config *StripeConfig) (*StripeProvider, error) {
	if config.SecretKey == "" {
		return nil, errors.New("Stripe SecretKey 不能为空")
	}
	
	// 设置 Stripe API 密钥
	stripe.Key = config.SecretKey
	
	// 可选：设置 API 基础地址
	if config.APIBase != "" {
		stripe.SetBackend(stripe.APIBackend, &stripe.BackendConfiguration{
			URL: config.APIBase,
		})
	}
	
	return &StripeProvider{
		config: config,
	}, nil
}

// GetName 获取支付提供商名称
func (p *StripeProvider) GetName() PaymentProvider {
	return ProviderStripe
}

// CreatePayment 创建 Stripe Checkout 会话
func (p *StripeProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 构建 Stripe Checkout 参数
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
			"alipay",
			"wechat_pay",
		}),
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(req.ReturnURL),
		CancelURL:  stripe.String(req.ReturnURL + "?cancelled=true"),
		ClientReferenceID: stripe.String(req.OrderID),
		Metadata: map[string]string{
			"user_id": req.UserID,
			"order_id": req.OrderID,
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(req.Currency),
					UnitAmount: stripe.Int64(req.Amount), // Stripe 使用最小货币单位
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(req.Description),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
	}
	
	// 设置过期时间（默认30分钟）
	params.ExpiresAt = stripe.Int64(time.Now().Add(30 * time.Minute).Unix())
	
	// 创建 Checkout Session
	session, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("创建 Stripe Checkout 会话失败: %w", err)
	}
	
	return &CreatePaymentResponse{
		PaymentID:  session.ID,
		Status:     PaymentStatusPending,
		PaymentURL: session.URL,
		ExpireAt:   time.Unix(session.ExpiresAt, 0),
	}, nil
}

// QueryPayment 查询支付状态
func (p *StripeProvider) QueryPayment(ctx context.Context, paymentID string) (*QueryPaymentResponse, error) {
	// 获取 Checkout Session
	session, err := session.Get(paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Stripe Session 失败: %w", err)
	}
	
	// 转换支付状态
	status := p.convertStatus(session.PaymentStatus)
	
	resp := &QueryPaymentResponse{
		PaymentID:    session.ID,
		OrderID:      session.ClientReferenceID,
		Status:       status,
		Amount:       session.AmountTotal,
		Currency:     string(session.Currency),
		Provider:     ProviderStripe,
		ThirdPartyID: session.PaymentIntent.ID,
		CreatedAt:    time.Unix(session.Created, 0),
		UpdatedAt:    time.Now(),
	}
	
	// 如果已支付，设置支付时间
	if status == PaymentStatusSuccess {
		paidAt := time.Unix(session.Created, 0)
		resp.PaidAt = &paidAt
	}
	
	return resp, nil
}

// CancelPayment 取消支付
func (p *StripeProvider) CancelPayment(ctx context.Context, paymentID string) error {
	// Stripe 不支持直接取消 Checkout Session
	// 可以通过过期机制自动取消
	// 这里只是记录取消操作
	return nil
}

// Refund 退款
func (p *StripeProvider) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// 首先查询支付信息获取 PaymentIntent ID
	payment, err := p.QueryPayment(ctx, req.PaymentID)
	if err != nil {
		return nil, err
	}
	
	if payment.Status != PaymentStatusSuccess {
		return nil, errors.New("只能对成功的支付进行退款")
	}
	
	// 创建退款
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(payment.ThirdPartyID),
		Reason:        stripe.String(string(stripe.RefundReasonRequestedByCustomer)),
		Metadata: map[string]string{
			"refund_id": req.RefundID,
			"reason":    req.Reason,
		},
	}
	
	// 如果指定了退款金额
	if req.Amount > 0 {
		params.Amount = stripe.Int64(req.Amount)
	}
	
	ref, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("创建退款失败: %w", err)
	}
	
	resp := &RefundResponse{
		RefundID: ref.ID,
		Status:   string(ref.Status),
		Amount:   ref.Amount,
	}
	
	if ref.Created > 0 {
		refundedAt := time.Unix(ref.Created, 0)
		resp.RefundedAt = &refundedAt
	}
	
	return resp, nil
}

// VerifyWebhook 验证 Stripe Webhook 签名
func (p *StripeProvider) VerifyWebhook(ctx context.Context, payload *WebhookPayload) (bool, error) {
	if p.config.WebhookSecret == "" {
		return false, errors.New("WebhookSecret 未配置")
	}
	
	// 签名验证在 ParseWebhook 中完成
	return true, nil
}

// ParseWebhook 解析 Stripe Webhook 数据
func (p *StripeProvider) ParseWebhook(ctx context.Context, body []byte, signature string) (*WebhookPayload, error) {
	if p.config.WebhookSecret == "" {
		return nil, errors.New("WebhookSecret 未配置")
	}
	
	// 验证签名并解析事件
	event, err := webhook.ConstructEvent(body, signature, p.config.WebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("验证 Webhook 签名失败: %w", err)
	}
	
	// 解析事件数据
	var data map[string]interface{}
	eventData, _ := json.Marshal(event.Data.Object)
	json.Unmarshal(eventData, &data)
	
	return &WebhookPayload{
		Provider:  ProviderStripe,
		EventType: string(event.Type),
		Data:      data,
		Signature: signature,
	}, nil
}

// convertStatus 转换 Stripe 支付状态为内部状态
func (p *StripeProvider) convertStatus(status stripe.CheckoutSessionPaymentStatus) PaymentStatus {
	switch status {
	case stripe.CheckoutSessionPaymentStatusPaid:
		return PaymentStatusSuccess
	case stripe.CheckoutSessionPaymentStatusUnpaid:
		return PaymentStatusPending
	case stripe.CheckoutSessionPaymentStatusNoPaymentRequired:
		return PaymentStatusSuccess
	default:
		return PaymentStatusPending
	}
}

// GetSession 获取 Checkout Session 详情（扩展方法）
func (p *StripeProvider) GetSession(sessionID string) (*stripe.CheckoutSession, error) {
	return session.Get(sessionID, nil)
}
