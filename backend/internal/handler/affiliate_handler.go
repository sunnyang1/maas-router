// Package handler 邀请返利处理器
// 提供邀请链接获取、返利记录查询、返利提现等功能
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"maas-router/internal/pkg/ctxkey"
	"maas-router/internal/service"
)

// AffiliateHandler 邀请返利处理器
type AffiliateHandler struct {
	AffiliateService service.AffiliateService
	Logger           *zap.Logger
}

// NewAffiliateHandler 创建邀请返利处理器
func NewAffiliateHandler(
	affiliateService service.AffiliateService,
	logger *zap.Logger,
) *AffiliateHandler {
	return &AffiliateHandler{
		AffiliateService: affiliateService,
		Logger:           logger,
	}
}

// GetInviteInfo 获取邀请信息
// GET /api/v1/user/affiliate/info
func (h *AffiliateHandler) GetInviteInfo(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	info, err := h.AffiliateService.GetInviteInfo(c.Request.Context(), userID.(int64))
	if err != nil {
		h.Logger.Error("获取邀请信息失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取邀请信息失败")
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetAffiliateStats 获取返利统计
// GET /api/v1/user/affiliate/stats
func (h *AffiliateHandler) GetAffiliateStats(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	stats, err := h.AffiliateService.GetAffiliateStats(c.Request.Context(), userID.(int64))
	if err != nil {
		h.Logger.Error("获取返利统计失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取返利统计失败")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetRebateRecords 获取返利记录
// GET /api/v1/user/affiliate/records
func (h *AffiliateHandler) GetRebateRecords(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	records, total, err := h.AffiliateService.GetRebateRecords(c.Request.Context(), userID.(int64), page, pageSize)
	if err != nil {
		h.Logger.Error("获取返利记录失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取返利记录失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetInviteRecords 获取邀请记录
// GET /api/v1/user/affiliate/invites
func (h *AffiliateHandler) GetInviteRecords(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	records, total, err := h.AffiliateService.GetInviteRecords(c.Request.Context(), userID.(int64), page, pageSize)
	if err != nil {
		h.Logger.Error("获取邀请记录失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取邀请记录失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// WithdrawRebateRequest 提现返利请求
type WithdrawRebateRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// WithdrawRebate 提现返利到账户余额
// POST /api/v1/user/affiliate/withdraw
func (h *AffiliateHandler) WithdrawRebate(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	var req WithdrawRebateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	err := h.AffiliateService.TransferRebateToBalance(c.Request.Context(), userID.(int64), req.Amount)
	if err != nil {
		h.Logger.Error("提现返利失败", zap.Error(err))
		ErrorResponse(c, http.StatusBadRequest, "WITHDRAW_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "提现成功",
		"amount":  req.Amount,
	})
}

// GenerateInviteCode 生成邀请码（如果没有）
// POST /api/v1/user/affiliate/code
func (h *AffiliateHandler) GenerateInviteCode(c *gin.Context) {
	userID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	inviteCode, err := h.AffiliateService.GenerateInviteCode(c.Request.Context(), userID.(int64))
	if err != nil {
		h.Logger.Error("生成邀请码失败", zap.Error(err))
		ErrorResponse(c, http.StatusInternalServerError, "GENERATE_FAILED", "生成邀请码失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invite_code": inviteCode,
	})
}
