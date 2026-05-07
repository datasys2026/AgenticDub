package router

import (
	"krillin-ai/internal/handler"
	"krillin-ai/static"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	api := r.Group("/api")

	hdl := handler.NewHandler()
	{
		api.POST("/capability/subtitleTask", hdl.StartSubtitleTask)
		api.GET("/capability/subtitleTask", hdl.GetSubtitleTask)
		api.POST("/file", hdl.UploadFile)
		api.GET("/file/*filepath", hdl.DownloadFile)
		api.HEAD("/file/*filepath", hdl.DownloadFile)
		api.GET("/config", hdl.GetConfig)
		api.POST("/config", hdl.UpdateConfig)

		// HITL Review APIs
		api.GET("/hitl/review/:task_id", hdl.GetReview)
		api.POST("/hitl/approve/:task_id", hdl.ApproveReview)
		api.POST("/hitl/reject/:task_id", hdl.RejectReview)
		api.GET("/hitl/status/:task_id", hdl.GetReviewStatus)
	}

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})
	r.StaticFS("/static", http.FS(static.EmbeddedFiles))
}
