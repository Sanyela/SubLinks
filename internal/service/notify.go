package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type IPInfo struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Org     string `json:"org"`
	AS      string `json:"as"`
}

type Notifier struct {
	botToken string
	chatID   string
	level    int
}

func NewNotifier(botToken, chatID string, level int) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		level:    level,
	}
}

func (n *Notifier) SendMessage(msgType, ip, additionalData string) error {
	if n.botToken == "" || n.chatID == "" {
		return nil
	}

	ipInfo, err := n.getIPInfo(ip)
	if err != nil {
		// 如果获取IP信息失败，仍然发送基本消息
		return n.sendTelegramMessage(fmt.Sprintf("%s\nIP: %s\n%s", msgType, ip, additionalData))
	}

	msg := fmt.Sprintf("%s\nIP: %s\n国家: %s\n城市: %s\n组织: %s\nASN: %s\n%s",
		msgType, ip, ipInfo.Country, ipInfo.City, ipInfo.Org, ipInfo.AS, additionalData)

	return n.sendTelegramMessage(msg)
}

func (n *Notifier) getIPInfo(ip string) (*IPInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IP API返回错误状态码: %d", resp.StatusCode)
	}

	var ipInfo IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return nil, err
	}

	return &ipInfo, nil
}

func (n *Notifier) sendTelegramMessage(message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	params := url.Values{}
	params.Set("chat_id", n.chatID)
	params.Set("parse_mode", "HTML")
	params.Set("text", message)

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return fmt.Errorf("发送Telegram消息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

func (n *Notifier) ShouldNotify(isSubscribeRequest bool) bool {
	return n.level == 1 || !isSubscribeRequest
}
