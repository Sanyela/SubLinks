package service

import (
	"fmt"
	"io"
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

	resp, err := http.Get(convertURL)
	if err != nil {
		return "", fmt.Errorf("转换请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("转换服务返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取转换结果失败: %w", err)
	}

	return string(body), nil
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
