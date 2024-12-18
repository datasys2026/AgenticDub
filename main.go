package main

import (
	"fmt"
	"krillin-ai/config"
	"krillin-ai/internal/router"

	"github.com/gin-gonic/gin"
)

type App struct {
	Engine *gin.Engine
}

func main() {
	var err error
	err = config.LoadConfig("./config/config.toml")
	if err != nil {
		panic(fmt.Sprintf("加载配置文件失败: %v", err))
	}
	app := App{
		Engine: gin.Default(),
	}
	app.Engine = gin.Default()
	router.SetupRouter(app.Engine)
	_ = app.Engine.Run(fmt.Sprintf("%s:%d", config.Conf.Server.Host, config.Conf.Server.Port))
}
