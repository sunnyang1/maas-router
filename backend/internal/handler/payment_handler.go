// Package handler 提供支付相关的 HTTP 处理器
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"maas-router/backend/internal/payment"
	"maas-router/backend/ent"
)

// PaymentHandler 支付处理器
type PaymentHandler struct {
	paymentService *payment.Service
	entClient      *ent.Client
}

// NewPaymentHandler 创建支付处理器
func NewPaymentHandler(paymentService *payment.Service, entClient *ent.Client) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		entClient:      entClient,
	}
}

// CreatePaymentRequest 创建支付请求（HTTP）
type CreatePaymentRequest struct {
	// 订单ID
	OrderID string `json:"order_id" binding:"required"`
	// 支付金额（单位：分）
	Amount int64 `json:"amount" binding:"required,gt=0"`
	// 货币代码
	Currency string `json:"currency" binding:"required"`
	// 支付提供商
	Provider string `json:"provider" binding:"required"`
	// 商品描述
	Description string `json:"description" binding:"required"`
	// 返回URL
	ReturnURL string `json:"return_url" binding:"required,url"`
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
	// 支付ID
	PaymentID string `json:"payment_id"`
	// 支付状态
	Status string `json:"status"`
	// 支付URL
	PaymentURL string `json:"payment_url,omitempty"`
	// 支付参数（用于JSAPI/APP支付）
	PaymentParams map[string]interface{} `json:"payment_params,omitempty"`
	// 过期时间
	ExpireAt time.Time `json:"expire_at"`
}

// CreatePayment 创建支付订单
// @Summary 创建支付订单
// @Description 创建一个新的支付订单
// @Tags 支付
// @Accept json
// @Produce json
// @Param request body CreatePaymentRequest true "支付请求参数"
// @Success 200 {object} CreatePaymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payments [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("请求参数错误: %v", err)})
		return
	}

	// 获取当前用户ID（从JWT中获取）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "未登录"})
		return
	}

	// 生成支付ID
	paymentID := generatePaymentID()

	// 构建回调URL
	notifyURL := fmt.Sprintf("%s/api/v1/payments/webhook/%s", getBaseURL(c), req.Provider)

	// 创建支付请求
	createReq := &payment.CreatePaymentRequest{
		OrderID:     req.OrderID,
		UserID:      userID.(string),
		Amount:      req.Amount,
		Currency:    req.Currency,
		Provider:    payment.PaymentProvider(req.Provider),
		Description: req.Description,
		NotifyURL:   notifyURL,
		ReturnURL:   req.ReturnURL,
		Metadata: map[string]string{
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		},
	}

	// 调用支付服务
	resp, err := h.paymentService.CreatePayment(c.Request.Context(), createReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("创建支付订单失败: %v", err)})
		return
	}

	// 保存支付订单到数据库
	_, err = h.entClient.PaymentOrder.Create().
		SetID(paymentID).
		SetOrderID(req.OrderID).
		SetUserID(userID.(string)).
		SetAmount(req.Amount).
		SetCurrency(req.Currency).
		SetProvider(req.Provider).
		SetStatus(string(resp.Status)).
		SetDescription(req.Description).
		SetPaymentURL(resp.PaymentURL).
		SetNotifyURL(notifyURL).
		SetReturnURL(req.ReturnURL).
		SetExpireAt(resp.ExpireAt).
		SetClientIP(c.ClientIP()).
		SetUserAgent(c.Request.UserAgent()).
		Save(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("保存支付订单失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, CreatePaymentResponse{
		PaymentID:     paymentID,
		Status:        string(resp.Status),
		PaymentURL:    resp.PaymentURL,
		PaymentParams: resp.PaymentParams,
		ExpireAt:      resp.ExpireAt,
	})
}

// QueryPaymentResponse 查询支付状态响应
type QueryPaymentResponse struct {
	// 支付ID
	PaymentID string `json:"payment_id"`
	// 订单ID
	OrderID string `json:"order_id"`
	// 支付状态
	Status string `json:"status"`
	// 支付金额
	Amount int64 `json:"amount"`
	// 货币代码
	Currency string `json:"currency"`
	// 支付提供商
	Provider string `json:"provider"`
	// 第三方支付单号
	ThirdPartyID string `json:"third_party_id,omitempty"`
	// 支付时间
	PaidAt *time.Time `json:"paid_at,omitempty"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// QueryPayment 查询支付状态
// @Summary 查询支付状态
// @Description 查询指定支付订单的状态
// @Tags 支付
// @Produce json
// @Param payment_id path string true "支付订单ID"
// @Success 200 {object} QueryPaymentResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/payments/{payment_id} [get]
func (h *PaymentHandler) QueryPayment(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "支付ID不能为空"})
		return
	}

	// 从数据库查询支付订单
	po, err := h.entClient.PaymentOrder.Get(c.Request.Context(), paymentID)
	if err != nil {
		if ent.IsNotFound(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "支付订单不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("查询支付订单失败: %v", err)})
		return
	}

	// 如果订单还在处理中，查询第三方支付状态
	if po.Status == string(payment.PaymentStatusPending) || po.Status == string(payment.PaymentStatusProcessing) {
		resp, err := h.paymentService.QueryPayment(c.Request.Context(), payment.PaymentProvider(po.Provider), po.ThirdPartyID)
		if err == nil && resp != nil {
			// 更新本地状态
			update := h.entClient.PaymentOrder.UpdateOneID(paymentID).
				SetStatus(string(resp.Status))

			if resp.Status == payment.PaymentStatusSuccess && resp.PaidAt != nil {
				update.SetPaidAt(*resp.PaidAt)
			}

			if resp.ThirdPartyID != "" {
				update.SetThirdPartyID(resp.ThirdPartyID)
			}

			updatedPO, err := update.Save(c.Request.Context())
			if err == nil {
				po = updatedPO
			}
		}
	}

	c.JSON(http.StatusOK, QueryPaymentResponse{
		PaymentID:    po.ID,
		OrderID:      po.OrderID,
		Status:       po.Status,
		Amount:       po.Amount,
		Currency:     po.Currency,
		Provider:     po.Provider,
		ThirdPartyID: po.ThirdPartyID,
		PaidAt:       po.PaidAt,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	})
}

// HandleWebhook 处理支付回调
// @Summary 处理支付回调
// @Description 接收第三方支付平台的异步通知
// @Tags 支付
// @Accept json,xml
// @Produce json
// @Param provider path string true "支付提供商"
// @Success 200 {object} map[string]string
// @Router /api/v1/payments/webhook/{provider} [post]
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "支付提供商不能为空"})
		return
	}

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "读取请求体失败"})
		return
	}

	// 获取签名
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		signature = c.GetHeader("X-Alipay-Signature")
	}
	if signature == "" {
		signature = c.Query("sign")
	}

	// 处理回调
	payload, err := h.paymentService.HandleWebhook(c.Request.Context(), payment.PaymentProvider(provider), body, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("处理回调失败: %v", err)})
		return
	}

	// 根据事件类型处理
	switch payload.EventType {
	case "payment_intent.succeeded", "charge.succeeded", "TRADE_SUCCESS", "payment.success":
		// 支付成功
		h.handlePaymentSuccess(c, payload)
	case "payment_intent.payment_failed", "charge.failed", "TRADE_CLOSED", "payment.fail":
		// 支付失败
		h.handlePaymentFailed(c, payload)
	case "charge.refunded", "TRADE_REFUND":
		// 退款
		h.handleRefund(c, payload)
	}

	// 返回成功响应
	switch provider {
	case "alipay":
		c.String(http.StatusOK, "success")
	case "wechat":
		c.XML(http.StatusOK, gin.H{
			"return_code": "SUCCESS",
			"return_msg":  "OK",
		})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

// handlePaymentSuccess 处理支付成功
func (h *PaymentHandler) handlePaymentSuccess(c *gin.Context, payload *payment.WebhookPayload) {
	// 从回调数据中提取订单信息
	var orderID, thirdPartyID string
	
	if data, ok := payload.Data["out_trade_no"].(string); ok {
		orderID = data
	} else if data, ok := payload.Data["client_reference_id"].(string); ok {
		orderID = data
	}
	
	if data, ok := payload.Data["trade_no"].(string); ok {
		thirdPartyID = data
	} else if data, ok := payload.Data["id"].(string); ok {
		thirdPartyID = data
	}

	if orderID == "" {
		return
	}

	// 更新支付订单状态
	_, err := h.entClient.PaymentOrder.Update().
		Where(ent.PaymentOrder.OrderIDEQ(orderID)).
		SetStatus(string(payment.PaymentStatusSuccess)).
		SetThirdPartyID(thirdPartyID).
		SetPaidAt(time.Now()).
		SetNotifyData(string(mustMarshal(payload.Data))).
		SetNotifiedAt(time.Now()).
		Save(c.Request.Context())

	if err != nil {
		// 记录错误日志
		fmt.Printf("更新支付订单状态失败: %v\n", err)
	}
}

// handlePaymentFailed 处理支付失败
func (h *PaymentHandler) handlePaymentFailed(c *gin.Context, payload *payment.WebhookPayload) {
	var orderID string
	var errorMessage string
	
	if data, ok := payload.Data["out_trade_no"].(string); ok {
		orderID = data
	}
	
	if data, ok := payload.Data["failure_message"].(string); ok {
		errorMessage = data
	} else if data, ok := payload.Data["err_code_des"].(string); ok {
		errorMessage = data
	}

	if orderID == "" {
		return
	}

	_, err := h.entClient.PaymentOrder.Update().
		Where(ent.PaymentOrder.OrderIDEQ(orderID)).
		SetStatus(string(payment.PaymentStatusFailed)).
		SetErrorMessage(errorMessage).
		SetNotifyData(string(mustMarshal(payload.Data))).
		SetNotifiedAt(time.Now()).
		Save(c.Request.Context())

	if err != nil {
		fmt.Printf("更新支付订单状态失败: %v\n", err)
	}
}

// handleRefund 处理退款
func (h *PaymentHandler) handleRefund(c *gin.Context, payload *payment.WebhookPayload) {
	// 处理退款逻辑
	// 可以在这里更新退款记录状态
}

// CancelPayment 取消支付订单
// @Summary 取消支付订单
// @Description 取消指定的支付订单
// @Tags 支付
// @Produce json
// @Param payment_id path string true "支付订单ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/payments/{payment_id}/cancel [post]
func (h *PaymentHandler) CancelPayment(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "支付ID不能为空"})
		return
	}

	// 查询支付订单
	po, err := h.entClient.PaymentOrder.Get(c.Request.Context(), paymentID)
	if err != nil {
		if ent.IsNotFound(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "支付订单不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("查询支付订单失败: %v", err)})
		return
	}

	// 检查订单状态
	if po.Status != string(payment.PaymentStatusPending) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "只能取消待支付的订单"})
		return
	}

	// 调用支付服务取消订单
	err = h.paymentService.CancelPayment(c.Request.Context(), payment.PaymentProvider(po.Provider), po.ThirdPartyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("取消支付订单失败: %v", err)})
		return
	}

	// 更新订单状态
	_, err = h.entClient.PaymentOrder.UpdateOneID(paymentID).
		SetStatus(string(payment.PaymentStatusCancelled)).
		Save(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("更新订单状态失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "支付订单已取消"})
}

// RefundRequest 退款请求
type RefundRequest struct {
	// 退款金额（单位：分），0表示全额退款
	Amount int64 `json:"amount"`
	// 退款原因
	Reason string `json:"reason" binding:"required"`
}

// RefundResponse 退款响应
type RefundResponse struct {
	// 退款ID
	RefundID string `json:"refund_id"`
	// 退款状态
	Status string `json:"status"`
	// 退款金额
	Amount int64 `json:"amount"`
}

// Refund 退款
// @Summary 退款
// @Description 对指定支付订单进行退款
// @Tags 支付
// @Accept json
// @Produce json
// @Param payment_id path string true "支付订单ID"
// @Param request body RefundRequest true "退款请求参数"
// @Success 200 {object} RefundResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/payments/{payment_id}/refund [post]
func (h *PaymentHandler) Refund(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "支付ID不能为空"})
		return
	}

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("请求参数错误: %v", err)})
		return
	}

	// 查询支付订单
	po, err := h.entClient.PaymentOrder.Get(c.Request.Context(), paymentID)
	if err != nil {
		if ent.IsNotFound(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "支付订单不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("查询支付订单失败: %v", err)})
		return
	}

	// 检查订单状态
	if po.Status != string(payment.PaymentStatusSuccess) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "只能对成功的支付订单进行退款"})
		return
	}

	// 确定退款金额
	refundAmount := req.Amount
	if refundAmount == 0 {
		refundAmount = po.Amount
	}

	// 生成退款单号
	refundID := generateRefundID()

	// 调用支付服务退款
	refundReq := &payment.RefundRequest{
		PaymentID: po.ThirdPartyID,
		Amount:    refundAmount,
		Reason:    req.Reason,
		RefundID:  refundID,
	}

	resp, err := h.paymentService.Refund(c.Request.Context(), payment.PaymentProvider(po.Provider), refundReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("退款失败: %v", err)})
		return
	}

	// 保存退款记录
	_, err = h.entClient.Refund.Create().
		SetID(uuid.New().String()).
		SetPaymentID(paymentID).
		SetRefundNo(refundID).
		SetAmount(refundAmount).
		SetReason(req.Reason).
		SetStatus(resp.Status).
		Save(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("保存退款记录失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, RefundResponse{
		RefundID: resp.RefundID,
		Status:   resp.Status,
		Amount:   resp.Amount,
	})
}

// ListPaymentsRequest 查询支付列表请求
type ListPaymentsRequest struct {
	// 页码
	Page int `form:"page" binding:"min=1"`
	// 每页数量
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	// 支付状态
	Status string `form:"status"`
	// 支付提供商
	Provider string `form:"provider"`
}

// ListPaymentsResponse 支付列表响应
type ListPaymentsResponse struct {
	// 总数
	Total int `json:"total"`
	// 数据列表
	List []QueryPaymentResponse `json:"list"`
}

// ListPayments 查询支付列表
// @Summary 查询支付列表
// @Description 查询当前用户的支付订单列表
// @Tags 支付
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param status query string false "支付状态"
// @Param provider query string false "支付提供商"
// @Success 200 {object} ListPaymentsResponse
// @Router /api/v1/payments [get]
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	var req ListPaymentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("请求参数错误: %v", err)})
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "未登录"})
		return
	}

	// 构建查询
	query := h.entClient.PaymentOrder.Query().
		Where(ent.PaymentOrder.UserIDEQ(userID.(string)))

	if req.Status != "" {
		query = query.Where(ent.PaymentOrder.StatusEQ(req.Status))
	}
	if req.Provider != "" {
		query = query.Where(ent.PaymentOrder.ProviderEQ(req.Provider))
	}

	// 查询总数
	total, err := query.Count(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("查询支付列表失败: %v", err)})
		return
	}

	// 查询数据
	orders, err := query.
		Order(ent.Desc(ent.PaymentOrder.FieldCreatedAt)).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		All(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("查询支付列表失败: %v", err)})
		return
	}

	// 转换响应
	list := make([]QueryPaymentResponse, len(orders))
	for i, order := range orders {
		list[i] = QueryPaymentResponse{
			PaymentID:    order.ID,
			OrderID:      order.OrderID,
			Status:       order.Status,
			Amount:       order.Amount,
			Currency:     order.Currency,
			Provider:     order.Provider,
			ThirdPartyID: order.ThirdPartyID,
			PaidAt:       order.PaidAt,
			CreatedAt:    order.CreatedAt,
			UpdatedAt:    order.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, ListPaymentsResponse{
		Total: total,
		List:  list,
	})
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// 辅助函数

// generatePaymentID 生成支付订单ID
func generatePaymentID() string {
	return "pay_" + uuid.New().String()
}

// generateRefundID 生成退款单号
func generateRefundID() string {
	return "ref_" + uuid.New().String()
}

// getBaseURL 获取基础URL
func getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// mustMarshal JSON 编码（忽略错误）
func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
