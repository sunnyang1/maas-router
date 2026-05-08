// Package payment 提供微信支付集成
package payment

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// WechatConfig 微信支付配置
type WechatConfig struct {
	// AppID 应用ID
	AppID string
	// MchID 商户号
	MchID string
	// APIKey API密钥
	APIKey string
	// AppSecret 应用密钥（用于获取OpenID）
	AppSecret string
	// NotifyURL 异步通知地址
	NotifyURL string
	// 证书路径（退款时需要）
	CertPath string
	KeyPath  string
}

// WechatProvider 微信支付提供商
type WechatProvider struct {
	config *WechatConfig
}

// NewWechatProvider 创建微信支付提供商
func NewWechatProvider(config *WechatConfig) (*WechatProvider, error) {
	if config.AppID == "" {
		return nil, errors.New("微信 AppID 不能为空")
	}
	if config.MchID == "" {
		return nil, errors.New("微信商户号不能为空")
	}
	if config.APIKey == "" {
		return nil, errors.New("微信 API 密钥不能为空")
	}
	
	return &WechatProvider{
		config: config,
	}, nil
}

// GetName 获取支付提供商名称
func (p *WechatProvider) GetName() PaymentProvider {
	return ProviderWechat
}

// WechatPayParams 微信支付参数（用于 JSAPI/APP 支付）
type WechatPayParams struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

// CreatePayment 创建微信支付订单
func (p *WechatProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 生成随机字符串
	nonceStr := p.generateNonceStr()
	
	// 构建统一下单请求参数
	params := map[string]string{
		"appid":            p.config.AppID,
		"mch_id":           p.config.MchID,
		"nonce_str":        nonceStr,
		"body":             req.Description,
		"out_trade_no":     req.OrderID,
		"total_fee":        fmt.Sprintf("%d", req.Amount),
		"spbill_create_ip": "127.0.0.1",
		"notify_url":       req.NotifyURL,
		"trade_type":       "NATIVE", // 扫码支付
		"product_id":       req.OrderID,
	}
	
	if req.NotifyURL == "" && p.config.NotifyURL != "" {
		params["notify_url"] = p.config.NotifyURL
	}
	
	// 生成签名
	sign := p.generateSign(params)
	params["sign"] = sign
	
	// 构建 XML 请求体
	xmlBody := p.mapToXML(params)
	
	// 这里应该发送HTTP请求到微信支付网关
	// 实际实现需要调用微信统一下单API
	// 为了演示，返回模拟数据
	
	// 构建支付参数
	payParams := &WechatPayParams{
		AppID:     p.config.AppID,
		TimeStamp: fmt.Sprintf("%d", time.Now().Unix()),
		NonceStr:  nonceStr,
		Package:   "prepay_id=wx" + p.generateNonceStr(),
		SignType:  "RSA",
	}
	
	return &CreatePaymentResponse{
		PaymentID:  req.OrderID,
		Status:     PaymentStatusPending,
		PaymentURL: "weixin://wxpay/bizpayurl?pr=" + p.generateNonceStr(), // 模拟支付URL
		PaymentParams: map[string]interface{}{
			"appId":     payParams.AppID,
			"timeStamp": payParams.TimeStamp,
			"nonceStr":  payParams.NonceStr,
			"package":   payParams.Package,
			"signType":  payParams.SignType,
			"paySign":   payParams.PaySign,
		},
		ExpireAt: time.Now().Add(2 * time.Hour), // 微信支付订单2小时过期
	}, nil
}

// QueryPayment 查询支付状态
func (p *WechatProvider) QueryPayment(ctx context.Context, paymentID string) (*QueryPaymentResponse, error) {
	// 构建查询请求参数
	params := map[string]string{
		"appid":        p.config.AppID,
		"mch_id":       p.config.MchID,
		"out_trade_no": paymentID,
		"nonce_str":    p.generateNonceStr(),
	}
	
	// 生成签名
	sign := p.generateSign(params)
	params["sign"] = sign
	
	// 这里应该发送HTTP请求到微信支付网关
	// 实际实现需要调用微信订单查询API
	
	return &QueryPaymentResponse{
		PaymentID:    paymentID,
		OrderID:      paymentID,
		Status:       PaymentStatusPending,
		Amount:       0,
		Currency:     "CNY",
		Provider:     ProviderWechat,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// CancelPayment 关闭订单
func (p *WechatProvider) CancelPayment(ctx context.Context, paymentID string) error {
	// 构建关闭订单请求参数
	params := map[string]string{
		"appid":        p.config.AppID,
		"mch_id":       p.config.MchID,
		"out_trade_no": paymentID,
		"nonce_str":    p.generateNonceStr(),
	}
	
	// 生成签名
	sign := p.generateSign(params)
	params["sign"] = sign
	
	// 这里应该发送HTTP请求到微信支付网关
	// 实际实现需要调用微信关闭订单API
	
	return nil
}

// Refund 退款
func (p *WechatProvider) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	// 构建退款请求参数
	params := map[string]string{
		"appid":         p.config.AppID,
		"mch_id":        p.config.MchID,
		"nonce_str":     p.generateNonceStr(),
		"out_trade_no":  req.PaymentID,
		"out_refund_no": req.RefundID,
		"total_fee":     fmt.Sprintf("%d", req.Amount),
		"refund_fee":    fmt.Sprintf("%d", req.Amount),
		"refund_desc":   req.Reason,
	}
	
	// 生成签名
	sign := p.generateSign(params)
	params["sign"] = sign
	
	// 这里应该发送HTTP请求到微信支付网关（需要使用证书）
	// 实际实现需要调用微信退款API
	
	return &RefundResponse{
		RefundID: req.RefundID,
		Status:   "SUCCESS",
		Amount:   req.Amount,
	}, nil
}

// VerifyWebhook 验证微信支付回调签名
func (p *WechatProvider) VerifyWebhook(ctx context.Context, payload *WebhookPayload) (bool, error) {
	// 微信支付回调是 XML 格式
	// 从 payload.Data 中提取参数
	data, ok := payload.Data["xml"].(map[string]interface{})
	if !ok {
		data = payload.Data
	}
	
	// 提取签名
	sign, ok := data["sign"].(string)
	if !ok {
		return false, errors.New("缺少签名")
	}
	
	// 转换为 string map
	params := make(map[string]string)
	for k, v := range data {
		if k == "sign" {
			continue
		}
		switch val := v.(type) {
		case string:
			params[k] = val
		default:
			params[k] = fmt.Sprintf("%v", val)
		}
	}
	
	// 验证签名
	expectedSign := p.generateSign(params)
	return subtle.ConstantTimeCompare([]byte(sign), []byte(expectedSign)) == 1, nil
}

// ParseWebhook 解析微信支付回调数据
func (p *WechatProvider) ParseWebhook(ctx context.Context, body []byte, signature string) (*WebhookPayload, error) {
	// 解析 XML
	var xmlData map[string]interface{}
	if err := xml.Unmarshal(body, &xmlData); err != nil {
		// 尝试 JSON 格式
		if err := json.Unmarshal(body, &xmlData); err != nil {
			return nil, fmt.Errorf("解析回调数据失败: %w", err)
		}
	}
	
	// 获取事件类型
	eventType := "unknown"
	if resultCode, ok := xmlData["result_code"].(string); ok {
		if resultCode == "SUCCESS" {
			eventType = "payment.success"
		} else {
			eventType = "payment.fail"
		}
	}
	
	return &WebhookPayload{
		Provider:  ProviderWechat,
		EventType: eventType,
		Data:      xmlData,
		Signature: signature,
	}, nil
}

// generateNonceStr 生成随机字符串（使用 crypto/rand）
func (p *WechatProvider) generateNonceStr() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		// fallback to timestamp-based nonce
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = letters[b[i]%byte(len(letters))]
	}
	return string(b)
}

// generateSign 生成签名
func (p *WechatProvider) generateSign(params map[string]string) string {
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
	content += "&key=" + p.config.APIKey
	
	// MD5 签名
	h := md5.New()
	h.Write([]byte(content))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// generateHMACSign 生成 HMAC-SHA256 签名
func (p *WechatProvider) generateHMACSign(params map[string]string) string {
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
	content += "&key=" + p.config.APIKey
	
	// HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(p.config.APIKey))
	h.Write([]byte(content))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// mapToXML 将 map 转换为 XML 字符串
func (p *WechatProvider) mapToXML(params map[string]string) string {
	type xmlMap struct {
		XMLName xml.Name `xml:"xml"`
		Items   []xmlItem
	}
	
	type xmlItem struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}
	
	var items []xmlItem
	for k, v := range params {
		items = append(items, xmlItem{
			XMLName: xml.Name{Local: k},
			Value:   v,
		})
	}
	
	xm := xmlMap{Items: items}
	data, _ := xml.Marshal(xm)
	return string(data)
}

// xmlToMap 将 XML 解析为 map
func (p *WechatProvider) xmlToMap(xmlData []byte) (map[string]string, error) {
	var result map[string]string
	if err := xml.Unmarshal(xmlData, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSandboxSignKey 获取沙箱密钥（测试用）
func (p *WechatProvider) GetSandboxSignKey() (string, error) {
	params := map[string]string{
		"mch_id":    p.config.MchID,
		"nonce_str": p.generateNonceStr(),
	}
	params["sign"] = p.generateSign(params)
	
	// 这里应该发送HTTP请求到微信支付沙箱网关
	// 实际实现需要调用获取沙箱密钥API
	
	return "sandbox_key", nil
}
