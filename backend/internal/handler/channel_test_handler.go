// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"maas-router/internal/service"
)

// ChannelTestHandler 渠道测试处理器
type ChannelTestHandler struct {
	ChannelTestService service.ChannelTestService
}

// NewChannelTestHandler 创建渠道测试处理器
func NewChannelTestHandler(channelTestService service.ChannelTestService) *ChannelTestHandler {
	return &ChannelTestHandler{
		ChannelTestService: channelTestService,
	}
}

// TestAccount 测试单个账户
// POST /api/v1/admin/accounts/:id/test
func (h *ChannelTestHandler) TestAccount(c *gin.Context) {
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

	result, err := h.ChannelTestService.TestAccount(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "TEST_FAILED",
				"message": "测试账户失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// TestAllAccounts 测试所有账户
// POST /api/v1/admin/accounts/test-all
func (h *ChannelTestHandler) TestAllAccounts(c *gin.Context) {
	results, err := h.ChannelTestService.TestAllAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "TEST_FAILED",
				"message": "测试所有账户失败: " + err.Error(),
			},
		})
		return
	}

	// 统计健康/不健康的数量
	healthy := 0
	unhealthy := 0
	for _, r := range results {
		if r.IsHealthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"results":   results,
			"total":     len(results),
			"healthy":   healthy,
			"unhealthy": unhealthy,
		},
	})
}

// GetTestResults 获取最新的测试结果
// GET /api/v1/admin/accounts/test-results
func (h *ChannelTestHandler) GetTestResults(c *gin.Context) {
	results := h.ChannelTestService.GetLatestResults()

	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"data":  []interface{}{},
			"total": 0,
			"message": "暂无测试结果，请先执行测试",
		})
		return
	}

	healthy := 0
	unhealthy := 0
	for _, r := range results {
		if r.IsHealthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"results":   results,
			"total":     len(results),
			"healthy":   healthy,
			"unhealthy": unhealthy,
		},
	})
}
