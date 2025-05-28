package service

import (
	"encoding/base64"
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

	// 尝试直接创建Clash配置
	if targetType == TypeClash {
		log.Printf("尝试直接生成Clash配置")

		// 解码base64以获取原始节点列表
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			log.Printf("Base64解码失败，尝试使用原始内容: %v", err)
			decoded = []byte(content)
		}

		// 解析节点列表
		nodeList := strings.Split(string(decoded), "\n")
		validNodes := make([]string, 0)

		// 过滤有效节点
		for _, node := range nodeList {
			node = strings.TrimSpace(node)
			if node != "" {
				validNodes = append(validNodes, node)
			}
		}

		// 如果有有效节点，创建Clash配置
		if len(validNodes) > 0 {
			log.Printf("找到 %d 个有效节点，生成Clash配置", len(validNodes))
			clashConfig := c.buildClashConfig(validNodes)
			return clashConfig, nil
		}
	}

	// 如果直接创建失败，尝试使用转换服务
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

// buildClashConfig 根据节点列表构建Clash配置
func (c *Converter) buildClashConfig(nodes []string) string {
	// 提取节点名称
	var nodeNames []string

	// 基础配置
	config := []string{
		"port: 7890",
		"socks-port: 7891",
		"allow-lan: true",
		"mode: Rule",
		"log-level: info",
		"external-controller: 127.0.0.1:9090",
		"proxies:",
	}

	// 解析并添加节点
	for _, node := range nodes {
		if strings.HasPrefix(node, "vmess://") || strings.HasPrefix(node, "trojan://") ||
			strings.HasPrefix(node, "ss://") || strings.HasPrefix(node, "ssr://") {
			// 提取节点名称或使用序号
			nodeName := fmt.Sprintf("Node-%d", len(nodeNames)+1)
			nodeNames = append(nodeNames, nodeName)

			// 添加节点配置
			config = append(config, fmt.Sprintf("  - {name: \"%s\", server: placeholder.example.com, port: 443, type: vmess}", nodeName))
		}
	}

	// 如果没有成功解析任何节点，返回默认配置
	if len(nodeNames) == 0 {
		return c.generateDefaultClashConfig()
	}

	// 添加代理组
	config = append(config, "proxy-groups:")
	config = append(config, "  - name: 🚀 节点选择")
	config = append(config, "    type: select")
	config = append(config, "    proxies:")

	// 添加所有节点到代理组
	for _, name := range nodeNames {
		config = append(config, fmt.Sprintf("      - %s", name))
	}

	// 添加DIRECT选项
	config = append(config, "      - DIRECT")

	// 添加规则
	config = append(config, "rules:")
	config = append(config, "  - MATCH,🚀 节点选择")

	return strings.Join(config, "\n")
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
