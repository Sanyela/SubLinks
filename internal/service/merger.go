package service

import (
	"bufio"
	"encoding/base64"
	"strings"
	"sync"

	"sublinks/config"
)

// NodeMerger 处理节点合并的服务
type NodeMerger struct {
	mainData string
}

// NewNodeMerger 创建新的节点合并服务
func NewNodeMerger(mainData string) *NodeMerger {
	return &NodeMerger{
		mainData: mainData,
	}
}

// MergeNodes 合并所有节点数据
func (m *NodeMerger) MergeNodes() (string, error) {
	// 处理主数据
	nodes := m.parseMainData()

	// 获取所有订阅URL
	urls := config.GetAllSubscribeURLs()

	// 并发获取订阅内容
	var wg sync.WaitGroup
	nodesChan := make(chan string, len(urls))

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if content, err := fetchSubscription(url); err == nil {
				nodesChan <- content
			}
		}(url)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(nodesChan)
	}()

	// 收集所有节点
	for content := range nodesChan {
		nodes = append(nodes, m.parseContent(content)...)
	}

	// 去重
	uniqueNodes := m.removeDuplicates(nodes)

	// 合并为最终结果
	result := strings.Join(uniqueNodes, "\n")
	return base64.StdEncoding.EncodeToString([]byte(result)), nil
}

// parseMainData 解析主数据
func (m *NodeMerger) parseMainData() []string {
	var nodes []string
	scanner := bufio.NewScanner(strings.NewReader(m.mainData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			nodes = append(nodes, line)
		}
	}
	return nodes
}

// parseContent 解析订阅内容
func (m *NodeMerger) parseContent(content string) []string {
	// 尝试base64解码
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		content = string(decoded)
	}

	var nodes []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			nodes = append(nodes, line)
		}
	}
	return nodes
}

// removeDuplicates 去除重复节点
func (m *NodeMerger) removeDuplicates(nodes []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, node := range nodes {
		if _, exists := seen[node]; !exists {
			seen[node] = struct{}{}
			result = append(result, node)
		}
	}
	return result
}

// fetchSubscription 获取订阅内容
func fetchSubscription(url string) (string, error) {
	// TODO: 实现HTTP请求获取订阅内容
	return "", nil
}
