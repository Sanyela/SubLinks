package service

import (
	"encoding/base64"
	"encoding/json"
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
	for i, node := range nodes {
		nodeConfig := ""
		nodeName := fmt.Sprintf("Node-%d", i+1)

		if strings.HasPrefix(node, "vmess://") {
			// 处理VMess节点
			vmessURL := strings.TrimPrefix(node, "vmess://")
			decodedBytes, err := base64.StdEncoding.DecodeString(vmessURL)
			if err == nil {
				var vmessInfo map[string]interface{}
				if err := json.Unmarshal(decodedBytes, &vmessInfo); err == nil {
					// 提取节点信息
					if name, ok := vmessInfo["ps"].(string); ok && name != "" {
						nodeName = name
					}

					// 构建VMess节点配置
					server, _ := vmessInfo["add"].(string)
					port, _ := vmessInfo["port"].(float64)
					id, _ := vmessInfo["id"].(string)
					aid, _ := vmessInfo["aid"].(float64)
					network, _ := vmessInfo["net"].(string)
					tls, _ := vmessInfo["tls"].(string)

					nodeConfig = fmt.Sprintf(`  - name: "%s"
    type: vmess
    server: %s
    port: %d
    uuid: %s
    alterId: %d
    cipher: auto
    network: %s
    tls: %t`,
						nodeName,
						server,
						int(port),
						id,
						int(aid),
						network,
						tls == "tls")

					config = append(config, nodeConfig)
					nodeNames = append(nodeNames, nodeName)
					continue
				}
			}
		} else if strings.HasPrefix(node, "ss://") {
			// 处理Shadowsocks节点
			ssURL := strings.TrimPrefix(node, "ss://")
			parts := strings.Split(ssURL, "@")
			if len(parts) >= 2 {
				methodAndPassword, err := base64.StdEncoding.DecodeString(parts[0])
				if err == nil {
					methodAndPasswordParts := strings.Split(string(methodAndPassword), ":")
					if len(methodAndPasswordParts) >= 2 {
						method := methodAndPasswordParts[0]
						password := methodAndPasswordParts[1]

						serverAndPort := parts[1]
						if hashIndex := strings.Index(serverAndPort, "#"); hashIndex > 0 {
							// 如果有名称标识，提取名称
							nodeName = serverAndPort[hashIndex+1:]
							serverAndPort = serverAndPort[:hashIndex]
						}

						serverParts := strings.Split(serverAndPort, ":")
						if len(serverParts) >= 2 {
							server := serverParts[0]
							port := serverParts[1]

							nodeConfig = fmt.Sprintf(`  - name: "%s"
    type: ss
    server: %s
    port: %s
    cipher: %s
    password: %s`,
								nodeName,
								server,
								port,
								method,
								password)

							config = append(config, nodeConfig)
							nodeNames = append(nodeNames, nodeName)
							continue
						}
					}
				}
			}
		} else if strings.HasPrefix(node, "trojan://") {
			// 处理Trojan节点
			trojanURL := strings.TrimPrefix(node, "trojan://")
			// 解析trojan URL
			uri, err := url.Parse("trojan://" + trojanURL)
			if err == nil {
				password := uri.User.String()
				host := uri.Host

				// 提取端口
				hostParts := strings.Split(host, ":")
				server := hostParts[0]
				port := "443"
				if len(hostParts) > 1 {
					port = hostParts[1]
				}

				// 提取节点名称
				if uri.Fragment != "" {
					nodeName = uri.Fragment
				}

				nodeConfig = fmt.Sprintf(`  - name: "%s"
    type: trojan
    server: %s
    port: %s
    password: %s
    sni: %s`,
					nodeName,
					server,
					port,
					password,
					server)

				config = append(config, nodeConfig)
				nodeNames = append(nodeNames, nodeName)
				continue
			}
		}

		// 如果无法解析，使用通用配置
		nodeConfig = fmt.Sprintf(`  - name: "%s"
    type: http
    server: example.com
    port: 443`, nodeName)

		config = append(config, nodeConfig)
		nodeNames = append(nodeNames, nodeName)
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
