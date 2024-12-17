package handler

import (
	"github.com/gin-gonic/gin"
	"krillin-ai/internal/dto"
	"krillin-ai/internal/response"
	"net/http"
)

func (h Handler) StartSubtitleTask(c *gin.Context) {
	var req dto.StartVideoSubtitleTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   "参数错误",
			Data:  nil,
		})
		return
	}

	svc := h.Service

	taskID, err := svc.StartSubtitleTask(req)
	if err != nil {
		response.R(c, response.Response{
			Error: -1,
			Msg:   err.Error(),
			Data:  nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": 0,
		"msg":   "成功",
		"data":  gin.H{"task_id": taskID},
	})
}

func GetSubtitleTaskStatus(c *gin.Context) {
	// todo
}
