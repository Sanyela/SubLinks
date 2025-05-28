package handler

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"sublinks/config"
	"sublinks/internal/service"
)

type Handler struct {
	merger    *service.NodeMerger
	converter *service.Converter
	notifier  *service.Notifier
	config    *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		merger:    service.NewNodeMerger(cfg.MainData),
		converter: service.NewConverter(cfg.Subconverter, cfg.SubConfig, cfg.SUBUpdateTime),
		notifier:  service.NewNotifier(cfg.TGBotToken, cfg.TGChatID, cfg.TGNotifyLevel),
		config:    cfg,
	}
}

// AddSubscribe 添加订阅
func (h *Handler) AddSubscribe(c *gin.Context) {
	// 验证token
	token := c.Query("token")
	if !h.validateToken(token, c.Request.URL.Path) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权的访问"})
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if err := config.AddSubscribeURL(req.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加订阅失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "订阅添加成功"})
}

// RemoveSubscribe 删除订阅
func (h *Handler) RemoveSubscribe(c *gin.Context) {
	// 验证token
	token := c.Query("token")
	if !h.validateToken(token, c.Request.URL.Path) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权的访问"})
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if err := config.RemoveSubscribeURL(req.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除订阅失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "订阅删除成功"})
}

// ListSubscribe 列出所有订阅
func (h *Handler) ListSubscribe(c *gin.Context) {
	// 验证token
	token := c.Query("token")
	if !h.validateToken(token, c.Request.URL.Path) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权的访问"})
		return
	}

	urls := config.GetAllSubscribeURLs()
	c.JSON(http.StatusOK, gin.H{"urls": urls})
}

func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	// 验证token
	token := r.URL.Query().Get("token")
	if !h.validateToken(token, r.URL.Path) {
		h.handleUnauthorized(w, r)
		return
	}

	// 打印客户端信息
	log.Printf("客户端请求订阅，UserAgent: %s", r.UserAgent())

	// 合并节点
	mergedContent, err := h.merger.MergeNodes()
	if err != nil {
		log.Printf("节点合并失败: %v", err)
		http.Error(w, "节点合并失败", http.StatusInternalServerError)
		return
	}

	if len(mergedContent) == 0 {
		log.Printf("警告: 合并后的节点内容为空")
	} else {
		log.Printf("合并后的节点内容大小: %d字节", len(mergedContent))
	}

	// 检测客户端类型
	clientType := h.converter.DetectClientType(r.UserAgent())
	log.Printf("检测到客户端类型: %s", clientType)

	// 浏览器直接访问时，应该解码base64
	if strings.Contains(strings.ToLower(r.UserAgent()), "mozilla") &&
		!strings.Contains(strings.ToLower(r.UserAgent()), "clash") &&
		!strings.Contains(strings.ToLower(r.UserAgent()), "v2ray") &&
		!strings.Contains(strings.ToLower(r.UserAgent()), "sing") {
		log.Printf("检测到浏览器访问，尝试解码base64内容")
		// 如果是浏览器直接访问，解码base64
		decoded, err := base64.StdEncoding.DecodeString(mergedContent)
		if err == nil {
			mergedContent = string(decoded)
			log.Printf("成功解码base64内容")
		} else {
			log.Printf("base64解码失败: %v", err)
		}
	}

	// 转换格式
	convertedContent, err := h.converter.Convert(mergedContent, clientType)
	if err != nil {
		log.Printf("订阅转换失败: %v", err)
		http.Error(w, "订阅转换失败", http.StatusInternalServerError)
		return
	}

	if len(convertedContent) < 10 {
		log.Printf("警告: 转换后的内容可能无效，长度太短: %d字节", len(convertedContent))
	} else {
		log.Printf("转换后的内容大小: %d字节", len(convertedContent))
	}

	// 设置响应头
	headers := h.converter.GetResponseHeaders(h.config.FileName)
	for key, value := range headers {
		w.Header().Set(key, value)
		log.Printf("设置响应头: %s=%s", key, value)
	}

	// 发送通知
	if h.notifier.ShouldNotify(true) {
		clientIP := r.Header.Get("CF-Connecting-IP")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		additionalData := "UA: " + r.UserAgent()
		h.notifier.SendMessage("#获取订阅", clientIP, additionalData)
	}

	// 返回结果d
	w.Write([]byte(convertedContent))
	log.Printf("成功返回订阅内容给客户端")
}

func (h *Handler) handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	if h.notifier.ShouldNotify(false) {
		clientIP := r.Header.Get("CF-Connecting-IP")
		additionalData := "UA: " + r.UserAgent()
		h.notifier.SendMessage("#异常访问", clientIP, additionalData)
	}

	// 返回nginx欢迎页
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Write([]byte(nginxWelcomePage))
}

func (h *Handler) validateToken(token, path string) bool {
	return token == h.config.MyToken ||
		strings.HasPrefix(path, "/"+h.config.MyToken) ||
		strings.Contains(path, "/"+h.config.MyToken+"?")
}

const nginxWelcomePage = `
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
`
