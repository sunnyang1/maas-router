// Package payment 提供支付宝支付集成
package payment

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	// AppID 应用ID
	AppID string
	// PrivateKey 应用私钥（RSA2）
	PrivateKey string
	// PublicKey 支付宝公钥
	PublicKey string
	// Gateway 网关地址，默认：https://openapi.alipay.com/gateway.do
	Gateway string
	// SignType 签名类型，默认：RSA2
	SignType string
	// Format 数据格式，默认：JSON
	Format string
	// Charset 编码格式，默认：UTF-8
	Charset string
	// NotifyURL 异步通知地址
	NotifyURL string
	// ReturnURL 同步返回地址
	ReturnURL string
}

// AlipayProvider 支付宝支付提供商
type AlipayProvider struct {
	config     *AlipayConfig
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewAlipayProvider 创建支付宝支付提供商
func NewAlipayProvider(config *AlipayConfig) (*AlipayProvider, error) {
	if config.AppID == "" {
		return nil, errors.New("支付宝 AppID 不能为空")
	}
	if config.PrivateKey == "" {
		return nil, errors.New("支付宝私钥不能为空")
	}
	if config.PublicKey == "" {
		return nil, errors.New("支付宝公钥不能为空")
	}
	
	// 设置默认值
	if config.Gateway == "" {
		config.Gateway = "https://openapi.alipay.com/gateway.do"
	}
	if config.SignType == "" {
		config.SignType = "RSA2"
	}
	if config.Format == "" {
		config.Format = "JSON"
	}
	if config.Charset == "" {
		config.Charset = "UTF-8"
	}
	
	provider := &AlipayProvider{
		config: config,
	}
	
	// 解析私钥
	privateKey, err := provider.parsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解析支付宝私钥失败: %w", err)
	}
	provider.privateKey = privateKey
	
	// 解析公钥
	publicKey, err := provider.parsePublicKey(config.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("解析支付宝公钥失败: %w", err)
	}
	provider.publicKey = publicKey
	
	return provider, nil
}

// GetName 获取支付提供商名称
func (p *AlipayProvider) GetName() PaymentProvider {
	return ProviderAlipay
}

// CreatePayment 创建支付宝支付订单
func (p *AlipayProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 构建请求参数
	bizContent := map[string]interface{}{
		"out_trade_no": req.OrderID,
		"total_amount": float64(req.Amount) / 100, // 转换为元
		"subject":      req.Description,
		"product_code": "FAST_INSTANT_TRADE_PAY",
		"timeout_express": "30m",
	}
	
	if req.ReturnURL != "" {
		bizContent["return_url"] = req.ReturnURL
	}
	if req.NotifyURL != "" {
		bizContent["notify_url"] = req.NotifyURL
	}
	
	// 构建公共参数
	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.page.pay",
		"format":      p.config.Format,
		"return_url":  req.ReturnURL,
		"notify_url":  req.NotifyURL,
		"charset":     p.config.Charset,
		"sign_type":   p.config.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": p.mustMarshal(bizContent),
	}
	
	// 生成签名
	sign, err := p.sign(params)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %w", err)
	}
	params["sign"] = sign
	
	// 构建支付URL
	paymentURL := p.buildURL(params)
	
	return &CreatePaymentResponse{
		PaymentID:  req.OrderID,
		Status:     PaymentStatusPending,
		PaymentURL: paymentURL,
		ExpireAt:   time.Now().Add(30 * time.Minute),
	}, nil
}

// QueryPayment 查询支付状态
func (p *AlipayProvider) QueryPayment(ctx context.Context, paymentID string) (*QueryPaymentResponse, error) {
	// 构建请求参数
	bizContent := map[string]interface{}{
		"out_trade_no": paymentID,
	}
	
	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.query",
		"format":      p.config.Format,
		"charset":     p.config.Charset,
		"sign_type":   p.config.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": p.mustMarshal(bizContent),
	}
	
	// 生成签名
	sign, err := p.sign(params)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %w", err)
	}
	params["sign"] = sign
	
	// 这里应该发送HTTP请求到支付宝网关
	// 为了演示，返回模拟数据
	// 实际实现需要调用支付宝API
	
	return &QueryPaymentResponse{
		PaymentID:    paymentID,
		OrderID:      paymentID,
		Status:       PaymentStatusPending,
		Amount:       0,
		Currency:     "CNY",
		Provider:     ProviderAlipay,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// CancelPayment 取消支付
func (p *AlipayProvider) CancelPayment(ctx context.Context, paymentID string) error {
	bizContent := map[string]interface{}{
		"out_trade_no": paymentID,
	}
	
	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.close",
		"format":      p.config.Format,
		"charset":     p.config.Charset,
		"sign_type":   p.config.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": p.mustMarshal(bizContent),
	}
	
	sign, err := p.sign(params)
	if err != nil {
		return fmt.Errorf("生成签名失败: %w", err)
	}
	params["sign"] = sign
	
	// 这里应该发送HTTP请求到支付宝网关
	return nil
}

// Refund 退款
func (p *AlipayProvider) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	bizContent := map[string]interface{}{
		"out_trade_no":   req.PaymentID,
		"refund_amount":  float64(req.Amount) / 100,
		"refund_reason":  req.Reason,
		"out_request_no": req.RefundID,
	}
	
	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.refund",
		"format":      p.config.Format,
		"charset":     p.config.Charset,
		"sign_type":   p.config.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": p.mustMarshal(bizContent),
	}
	
	sign, err := p.sign(params)
	if err != nil {
		return nil, fmt.Errorf("生成签名失败: %w", err)
	}
	params["sign"] = sign
	
	// 这里应该发送HTTP请求到支付宝网关
	return &RefundResponse{
		RefundID: req.RefundID,
		Status:   "SUCCESS",
		Amount:   req.Amount,
	}, nil
}

// VerifyWebhook 验证支付宝回调签名
func (p *AlipayProvider) VerifyWebhook(ctx context.Context, payload *WebhookPayload) (bool, error) {
	// 从 payload.Data 中提取参数
	data, ok := payload.Data["data"].(map[string]interface{})
	if !ok {
		return false, errors.New("无效的回调数据")
	}
	
	// 提取签名
	sign, ok := payload.Data["sign"].(string)
	if !ok {
		return false, errors.New("缺少签名")
	}
	
	// 验证签名
	return p.verifySign(data, sign)
}

// ParseWebhook 解析支付宝回调数据
func (p *AlipayProvider) ParseWebhook(ctx context.Context, body []byte, signature string) (*WebhookPayload, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析回调数据失败: %w", err)
	}
	
	return &WebhookPayload{
		Provider:  ProviderAlipay,
		EventType: p.getEventType(data),
		Data:      data,
		Signature: signature,
	}, nil
}

// sign 生成签名
func (p *AlipayProvider) sign(params map[string]string) (string, error) {
	// 过滤空值和 sign 字段
	filtered := make(map[string]string)
	for k, v := range params {
		if k != "sign" && v != "" {
			filtered[k] = v
		}
	}
	
	// 按键排序
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// 拼接字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, filtered[k]))
	}
	content := strings.Join(parts, "&")
	
	// RSA2 签名
	hash := sha256Hash(content)
	signature, err := rsa.SignPKCS1v15(nil, p.privateKey, crypto.SHA256, hash)
	if err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifySign 验证签名
func (p *AlipayProvider) verifySign(data map[string]interface{}, sign string) (bool, error) {
	// 将 map 转换为 string map
	params := make(map[string]string)
	for k, v := range data {
		if k == "sign" || k == "sign_type" {
			continue
		}
		switch val := v.(type) {
		case string:
			params[k] = val
		default:
			params[k] = fmt.Sprintf("%v", val)
		}
	}
	
	// 按键排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// 拼接字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	content := strings.Join(parts, "&")
	
	// 解码签名
	signature, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false, err
	}
	
	// 验证签名
	hash := sha256Hash(content)
	err = rsa.VerifyPKCS1v15(p.publicKey, crypto.SHA256, hash, signature)
	return err == nil, nil
}

// buildURL 构建请求URL
func (p *AlipayProvider) buildURL(params map[string]string) string {
	u, _ := url.Parse(p.config.Gateway)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// parsePrivateKey 解析 RSA 私钥
func (p *AlipayProvider) parsePrivateKey(key string) (*rsa.PrivateKey, error) {
	// 处理 PEM 格式
	key = strings.ReplaceAll(key, "\\n", "\n")
	if !strings.HasPrefix(key, "-----") {
		key = "-----BEGIN RSA PRIVATE KEY-----\n" + key + "\n-----END RSA PRIVATE KEY-----"
	}
	
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, errors.New("无法解析私钥")
	}
	
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 尝试 PKCS8 格式
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		var ok bool
		privateKey, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("私钥类型错误")
		}
	}
	
	return privateKey, nil
}

// parsePublicKey 解析 RSA 公钥
func (p *AlipayProvider) parsePublicKey(key string) (*rsa.PublicKey, error) {
	// 处理 PEM 格式
	key = strings.ReplaceAll(key, "\\n", "\n")
	if !strings.HasPrefix(key, "-----") {
		key = "-----BEGIN PUBLIC KEY-----\n" + key + "\n-----END PUBLIC KEY-----"
	}
	
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, errors.New("无法解析公钥")
	}
	
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	
	publicKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("公钥类型错误")
	}
	
	return publicKey, nil
}

// getEventType 获取事件类型
func (p *AlipayProvider) getEventType(data map[string]interface{}) string {
	if method, ok := data["method"].(string); ok {
		return method
	}
	if tradeStatus, ok := data["trade_status"].(string); ok {
		return tradeStatus
	}
	return "unknown"
}

// mustMarshal JSON 编码
func (p *AlipayProvider) mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// sha256Hash 计算 SHA256 哈希
func sha256Hash(content string) []byte {
	h := sha256.Sum256([]byte(content))
	return h[:]
}
