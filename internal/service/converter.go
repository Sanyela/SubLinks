package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type ConverterType string

const (
	TypeV2ray   ConverterType = "v2ray"
	TypeClash   ConverterType = "clash"
	TypeSingBox ConverterType = "singbox"
)

type Converter struct {
	backend    string
	configFile string
	updateTime int
}

func NewConverter(backend, configFile string, updateTime int) *Converter {
	return &Converter{
		backend:    backend,
		configFile: configFile,
		updateTime: updateTime,
	}
}

func (c *Converter) Convert(content string, targetType ConverterType) (string, error) {
	if targetType == TypeV2ray {
		return content, nil
	}

	params := url.Values{}
	params.Set("target", string(targetType))
	params.Set("url", content)
	params.Set("insert", "false")
	params.Set("config", c.configFile)
	params.Set("emoji", "true")
	params.Set("list", "false")
	params.Set("tfo", "false")
	params.Set("scv", "true")
	params.Set("fdn", "false")
	params.Set("sort", "false")
	params.Set("new_name", "true")

	convertURL := fmt.Sprintf("https://%s/sub?%s", c.backend, params.Encode())

	log.Printf("转换请求URL: %s", convertURL)
	log.Printf("转换目标类型: %s", targetType)

	resp, err := http.Get(convertURL)
	if err != nil {
		log.Printf("转换请求失败: %v", err)
		if targetType == TypeClash {
			return c.generateDefaultClashConfig(), nil
		}
		return "", fmt.Errorf("转换请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("转换服务返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
		if targetType == TypeClash {
			log.Printf("Clash转换失败，返回默认配置")
			return c.generateDefaultClashConfig(), nil
		}
		return "", fmt.Errorf("转换服务返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取转换结果失败: %v", err)
		if targetType == TypeClash {
			return c.generateDefaultClashConfig(), nil
		}
		return "", fmt.Errorf("读取转换结果失败: %w", err)
	}

	if len(body) < 10 && targetType == TypeClash {
		log.Printf("转换结果内容过短，可能无效，返回默认配置")
		return c.generateDefaultClashConfig(), nil
	}

	return string(body), nil
}

func (c *Converter) generateDefaultClashConfig() string {
	return `
port: 7890
socks-port: 7891
allow-lan: true
mode: Rule
log-level: info
external-controller: 127.0.0.1:9090
proxies:
  - name: 默认节点
    type: http
    server: example.com
    port: 443
    username: username
    password: password
    tls: true
proxy-groups:
  - name: 🚀 节点选择
    type: select
    proxies:
      - 默认节点
      - DIRECT
rules:
  - MATCH,🚀 节点选择
`
}

func (c *Converter) DetectClientType(userAgent string) ConverterType {
	userAgent = strings.ToLower(userAgent)

	switch {
	case strings.Contains(userAgent, "clash") && !strings.Contains(userAgent, "nekobox"):
		return TypeClash
	case strings.Contains(userAgent, "sing-box") || strings.Contains(userAgent, "singbox"):
		return TypeSingBox
	default:
		return TypeV2ray
	}
}

func (c *Converter) GetResponseHeaders(filename string) map[string]string {
	return map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename*=utf-8''%s; filename=%s`,
			url.QueryEscape(filename), filename),
		"content-type":            "text/plain; charset=utf-8",
		"Profile-Update-Interval": fmt.Sprintf("%d", c.updateTime),
	}
}
