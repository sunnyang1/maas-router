// Package handler 提供 MaaS-Router 的 HTTP 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"maas-router/internal/config"
)

// ConfigHandler 配置处理器，返回前端所需的运行时配置
type ConfigHandler struct {
	Config *config.Config
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{Config: cfg}
}

// GetPublicConfig 返回前端公开配置
// GET /api/v1/public/config
func (h *ConfigHandler) GetPublicConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"apiUrl":  h.getBaseURL(),
		"version": "1.1.0",
	})
}

// getBaseURL 获取 API 基础 URL
// 优先使用环境变量 MAAS_ROUTER_PUBLIC_BASE_URL，否则根据服务器配置构建
func (h *ConfigHandler) getBaseURL() string {
	// 检查是否有配置的外部访问地址
	if h.Config.Server.Host != "" && h.Config.Server.Host != "0.0.0.0" {
		return "http://" + h.Config.Server.Host + ":" + string(rune(h.Config.Server.Port))
	}
	// 默认返回相对路径，让前端使用同域请求
	return ""
}
