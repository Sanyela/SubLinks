package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"sublinks/config"
	"sublinks/internal/handler"
)

func main() {
	// 初始化配置
	if err := initConfig(); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}

	// 加载动态订阅
	if err := config.LoadDynamicSubscribe(); err != nil {
		log.Printf("加载动态订阅失败: %v", err)
	}

	// 启动订阅文件监视
	if err := config.WatchSubscribeFile(); err != nil {
		log.Printf("启动订阅文件监视失败: %v", err)
	}

	// 创建处理器
	h := handler.NewHandler(&config.GlobalConfig)

	// 设置路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// API路由组
	api := r.Group("/api")
	{
		// 订阅管理
		api.POST("/subscribe", h.AddSubscribe)      // 添加订阅
		api.DELETE("/subscribe", h.RemoveSubscribe) // 删除订阅
		api.GET("/subscribe", h.ListSubscribe)      // 列出所有订阅
	}

	// 主路由处理订阅请求
	r.GET("/*path", func(c *gin.Context) {
		h.HandleSubscribe(c.Writer, c.Request)
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("服务器启动在 :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func initConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("my_token", "auto")
	viper.SetDefault("file_name", "Pages-SUB-Convert")
	viper.SetDefault("sub_update_time", 6)
	viper.SetDefault("subconverter", "apiurl.v1.mk")
	viper.SetDefault("sub_config", "https://raw.githubusercontent.com/cmliu/ACL4SSR/main/Clash/config/ACL4SSR_Online_MultiCountry.ini")
	viper.SetDefault("subscribe_file", "subscribe.json")

	// 从环境变量读取配置
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SUB")

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// 解析配置到结构体
	return viper.Unmarshal(&config.GlobalConfig)
}
