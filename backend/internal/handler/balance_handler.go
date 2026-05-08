// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"maas-router/internal/service"
)

// BalanceHandler 余额查询处理器
type BalanceHandler struct {
	BalanceService service.BalanceService
}

// NewBalanceHandler 创建余额查询处理器
func NewBalanceHandler(balanceService service.BalanceService) *BalanceHandler {
	return &BalanceHandler{
		BalanceService: balanceService,
	}
}

// GetBalance 获取单个账户余额
// GET /api/v1/admin/accounts/:id/balance
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "缺少账户ID",
			},
		})
		return
	}

	info, err := h.BalanceService.GetCachedBalance(accountID)
	if err != nil {
		// 缓存未命中，实时查询
		info, err = h.BalanceService.QueryBalance(c.Request.Context(), accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "QUERY_FAILED",
					"message": "查询余额失败: " + err.Error(),
				},
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": info,
	})
}

// GetAllBalances 获取所有账户余额
// GET /api/v1/admin/accounts/balances
func (h *BalanceHandler) GetAllBalances(c *gin.Context) {
	results, err := h.BalanceService.QueryAllBalances(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "QUERY_FAILED",
				"message": "查询所有余额失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": len(results),
	})
}

// RefreshBalance 强制刷新单个账户余额
// POST /api/v1/admin/accounts/:id/balance/refresh
func (h *BalanceHandler) RefreshBalance(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "缺少账户ID",
			},
		})
		return
	}

	info, err := h.BalanceService.QueryBalance(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "REFRESH_FAILED",
				"message": "刷新余额失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    info,
		"message": "余额已刷新",
	})
}
