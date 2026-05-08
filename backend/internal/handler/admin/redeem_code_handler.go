// Package admin 管理员接口处理器
// 提供卡密管理相关功能
package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"maas-router/internal/handler"
	"maas-router/internal/pkg/ctxkey"
	"maas-router/internal/service"
)

// RedeemCodeHandler 卡密管理处理器
type RedeemCodeHandler struct {
	RedeemCodeService service.RedeemCodeService
	Logger            *zap.Logger
}

// NewRedeemCodeHandler 创建卡密管理处理器
func NewRedeemCodeHandler(
	redeemCodeService service.RedeemCodeService,
	logger *zap.Logger,
) *RedeemCodeHandler {
	return &RedeemCodeHandler{
		RedeemCodeService: redeemCodeService,
		Logger:            logger,
	}
}

// GenerateCodesRequest 生成卡密请求
type GenerateCodesRequest struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Count     int     `json:"count" binding:"required,min=1,max=1000"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	Remark    string  `json:"remark,omitempty"`
}

// GenerateCodes 批量生成卡密
// POST /api/v1/admin/redeem-codes/generate
func (h *RedeemCodeHandler) GenerateCodes(c *gin.Context) {
	adminID, exists := c.Get(string(ctxkey.ContextKeyUserID))
	if !exists {
		handler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "未授权")
		return
	}

	var req GenerateCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	// 解析过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			handler.ErrorResponse(c, http.StatusBadRequest, "INVALID_EXPIRES_AT", "过期时间格式错误")
			return
		}
		expiresAt = &t
	}

	codes, err := h.RedeemCodeService.GenerateCodes(
		c.Request.Context(),
		adminID.(int64),
		req.Amount,
		req.Count,
		expiresAt,
		req.Remark,
	)
	if err != nil {
		h.Logger.Error("生成卡密失败", zap.Error(err))
		handler.ErrorResponse(c, http.StatusInternalServerError, "GENERATE_FAILED", err.Error())
		return
	}

	// 提取卡密信息
	codeList := make([]gin.H, 0, len(codes))
	for _, code := range codes {
		codeList = append(codeList, gin.H{
			"id":         code.ID,
			"code":       code.Code,
			"amount":     code.Amount,
			"status":     code.Status,
			"batch_no":   code.BatchNo,
			"expires_at": code.ExpiresAt,
			"created_at": code.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "生成成功",
		"batch_no":  codes[0].BatchNo,
		"count":     len(codes),
		"codes":     codeList,
	})
}

// ListCodes 获取卡密列表
// GET /api/v1/admin/redeem-codes
func (h *RedeemCodeHandler) ListCodes(c *gin.Context) {
	status := c.Query("status")
	batchNo := c.Query("batch_no")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	codes, total, err := h.RedeemCodeService.GetCodeList(c.Request.Context(), status, batchNo, page, pageSize)
	if err != nil {
		h.Logger.Error("获取卡密列表失败", zap.Error(err))
		handler.ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取卡密列表失败")
		return
	}

	// 构建响应
	codeList := make([]gin.H, 0, len(codes))
	for _, code := range codes {
		item := gin.H{
			"id":         code.ID,
			"code":       code.Code,
			"amount":     code.Amount,
			"status":     code.Status,
			"batch_no":   code.BatchNo,
			"remark":     code.Remark,
			"created_at": code.CreatedAt,
		}
		if code.ExpiresAt != nil {
			item["expires_at"] = code.ExpiresAt
		}
		if code.UsedAt != nil {
			item["used_at"] = code.UsedAt
			item["used_by"] = code.UsedBy
		}
		codeList = append(codeList, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  codeList,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetCodeDetail 获取卡密详情
// GET /api/v1/admin/redeem-codes/:id
func (h *RedeemCodeHandler) GetCodeDetail(c *gin.Context) {
	codeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		handler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "无效的卡密ID")
		return
	}

	code, err := h.RedeemCodeService.GetCodeDetail(c.Request.Context(), codeID)
	if err != nil {
		h.Logger.Error("获取卡密详情失败", zap.Error(err))
		handler.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "卡密不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         code.ID,
		"code":       code.Code,
		"amount":     code.Amount,
		"status":     code.Status,
		"batch_no":   code.BatchNo,
		"remark":     code.Remark,
		"expires_at": code.ExpiresAt,
		"used_at":    code.UsedAt,
		"used_by":    code.UsedBy,
		"created_at": code.CreatedAt,
	})
}

// DisableCode 禁用卡密
// PUT /api/v1/admin/redeem-codes/:id/disable
func (h *RedeemCodeHandler) DisableCode(c *gin.Context) {
	codeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		handler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "无效的卡密ID")
		return
	}

	err = h.RedeemCodeService.DisableCode(c.Request.Context(), codeID)
	if err != nil {
		h.Logger.Error("禁用卡密失败", zap.Error(err))
		handler.ErrorResponse(c, http.StatusBadRequest, "DISABLE_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "禁用成功",
	})
}

// ExportCodes 导出卡密
// GET /api/v1/admin/redeem-codes/export
func (h *RedeemCodeHandler) ExportCodes(c *gin.Context) {
	batchNo := c.Query("batch_no")
	if batchNo == "" {
		handler.ErrorResponse(c, http.StatusBadRequest, "MISSING_BATCH_NO", "缺少批次号")
		return
	}

	codes, err := h.RedeemCodeService.ExportCodes(c.Request.Context(), batchNo)
	if err != nil {
		h.Logger.Error("导出卡密失败", zap.Error(err))
		handler.ErrorResponse(c, http.StatusInternalServerError, "EXPORT_FAILED", err.Error())
		return
	}

	// 构建CSV内容
	var csvContent string
	csvContent = "卡密,面额,状态,创建时间\n"
	for _, code := range codes {
		csvContent += code.Code + "," +
			strconv.FormatFloat(code.Amount, 'f', 2, 64) + "," +
			string(code.Status) + "," +
			code.CreatedAt.Format("2006-01-02 15:04:05") + "\n"
	}

	// 设置响应头
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=redeem_codes_"+batchNo+".csv")
	c.String(http.StatusOK, csvContent)
}
