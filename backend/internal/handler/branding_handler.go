// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"maas-router/internal/service"
)

// BrandingHandler 品牌设置处理器
type BrandingHandler struct {
	BrandingService service.BrandingService
}

// NewBrandingHandler 创建品牌设置处理器
func NewBrandingHandler(brandingService service.BrandingService) *BrandingHandler {
	return &BrandingHandler{
		BrandingService: brandingService,
	}
}

// GetBranding 获取品牌设置（管理员）
// GET /api/v1/admin/branding
func (h *BrandingHandler) GetBranding(c *gin.Context) {
	settings, err := h.BrandingService.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取品牌设置失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}

// UpdateBranding 更新品牌设置（管理员）
// PUT /api/v1/admin/branding
func (h *BrandingHandler) UpdateBranding(c *gin.Context) {
	var settings service.BrandingSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "请求参数无效: " + err.Error(),
			},
		})
		return
	}

	if err := h.BrandingService.UpdateSettings(c.Request.Context(), &settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "UPDATE_FAILED",
				"message": "更新品牌设置失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    settings,
		"message": "品牌设置已更新",
	})
}

// GetPublicBranding 获取公开的品牌设置（前端使用）
// GET /api/v1/branding
func (h *BrandingHandler) GetPublicBranding(c *gin.Context) {
	settings, err := h.BrandingService.GetPublicSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "获取品牌设置失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}
