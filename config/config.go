package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

type Config struct {
	// 基本配置
	MyToken       string `mapstructure:"my_token" json:"my_token"`
	FileName      string `mapstructure:"file_name" json:"file_name"`
	SUBUpdateTime int    `mapstructure:"sub_update_time" json:"sub_update_time"`

	// Telegram配置
	TGBotToken    string `mapstructure:"tg_bot_token" json:"tg_bot_token"`
	TGChatID      string `mapstructure:"tg_chat_id" json:"tg_chat_id"`
	TGNotifyLevel int    `mapstructure:"tg_notify_level" json:"tg_notify_level"`

	// 订阅转换配置
	Subconverter string `mapstructure:"subconverter" json:"subconverter"`
	SubConfig    string `mapstructure:"sub_config" json:"sub_config"`

	// 节点数据
	MainData      string   `mapstructure:"main_data" json:"main_data"`
	SubscribeURLs []string `mapstructure:"subscribe_urls" json:"subscribe_urls"`
	WarpConfig    string   `mapstructure:"warp_config" json:"warp_config"`

	// 动态订阅文件路径
	SubscribeFile string `mapstructure:"subscribe_file" json:"subscribe_file"`
}

type DynamicSubscribe struct {
	URLs []string `json:"urls"`
}

var (
	GlobalConfig Config
	dynamicMutex sync.RWMutex
	dynamicURLs  []string
)

// LoadDynamicSubscribe 加载动态订阅
func LoadDynamicSubscribe() error {
	dynamicMutex.Lock()
	defer dynamicMutex.Unlock()

	if GlobalConfig.SubscribeFile == "" {
		GlobalConfig.SubscribeFile = "subscribe.json"
	}

	// 确保文件存在
	if _, err := os.Stat(GlobalConfig.SubscribeFile); os.IsNotExist(err) {
		// 创建默认文件
		defaultSub := DynamicSubscribe{URLs: []string{}}
		data, _ := json.MarshalIndent(defaultSub, "", "    ")
		if err := ioutil.WriteFile(GlobalConfig.SubscribeFile, data, 0644); err != nil {
			return err
		}
	}

	// 读取文件
	data, err := ioutil.ReadFile(GlobalConfig.SubscribeFile)
	if err != nil {
		return err
	}

	var sub DynamicSubscribe
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}

	dynamicURLs = sub.URLs
	return nil
}

// GetAllSubscribeURLs 获取所有订阅URL
func GetAllSubscribeURLs() []string {
	dynamicMutex.RLock()
	defer dynamicMutex.RUnlock()

	// 合并静态和动态订阅
	allURLs := make([]string, 0, len(GlobalConfig.SubscribeURLs)+len(dynamicURLs))
	allURLs = append(allURLs, GlobalConfig.SubscribeURLs...)
	allURLs = append(allURLs, dynamicURLs...)
	return allURLs
}

// AddSubscribeURL 添加新的订阅URL
func AddSubscribeURL(url string) error {
	dynamicMutex.Lock()
	defer dynamicMutex.Unlock()

	// 检查URL是否已存在
	for _, existingURL := range dynamicURLs {
		if existingURL == url {
			return nil
		}
	}

	// 添加新URL
	dynamicURLs = append(dynamicURLs, url)

	// 保存到文件
	sub := DynamicSubscribe{URLs: dynamicURLs}
	data, err := json.MarshalIndent(sub, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(GlobalConfig.SubscribeFile, data, 0644)
}

// RemoveSubscribeURL 移除订阅URL
func RemoveSubscribeURL(url string) error {
	dynamicMutex.Lock()
	defer dynamicMutex.Unlock()

	// 查找并移除URL
	newURLs := make([]string, 0, len(dynamicURLs))
	for _, existingURL := range dynamicURLs {
		if existingURL != url {
			newURLs = append(newURLs, existingURL)
		}
	}

	dynamicURLs = newURLs

	// 保存到文件
	sub := DynamicSubscribe{URLs: dynamicURLs}
	data, err := json.MarshalIndent(sub, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(GlobalConfig.SubscribeFile, data, 0644)
}

// WatchSubscribeFile 监视订阅文件变化
func WatchSubscribeFile() error {
	// 启动文件监视
	go func() {
		for {
			// 每30秒检查一次文件变化
			time.Sleep(30 * time.Second)
			if err := LoadDynamicSubscribe(); err != nil {
				log.Printf("重新加载订阅文件失败: %v", err)
			}
		}
	}()

	return nil
}

func Init() error {
	// TODO: 从环境变量或配置文件加载配置
	return nil
}
