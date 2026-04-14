package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/infrastructure/opencode"
)

// ListOpenCodeModels 返回 opencode CLI 支持的模型列表
func ListOpenCodeModels(c *gin.Context) {
	models, err := opencode.ListModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{"models": models})
}
