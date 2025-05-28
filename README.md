# SubLinks

SubLinks 是一个强大的订阅链接管理器，支持多种客户端格式转换（V2ray、Clash、SingBox）和动态订阅管理。

## 特性

- 🚀 支持多种客户端格式（V2ray、Clash、SingBox）
- 📱 自动识别客户端类型
- 🔄 动态订阅管理（支持热加载）
- 🔔 Telegram 通知支持
- 🔒 Token 访问控制
- 💻 跨平台支持（Windows、Linux）

## 快速开始

1. 从 [Releases](https://github.com/yourusername/sublinks/releases) 下载适合您系统的版本
2. 解压下载的文件
3. 将 `config.yaml.example` 重命名为 `config.yaml` 并修改配置
4. 运行程序：
   - Windows: 双击 `sublinks.exe`
   - Linux: `./sublinks`

## 配置说明

配置文件 `config.yaml` 示例：

```yaml
# 基本配置
my_token: "your_token_here"        # 访问令牌，用于验证请求
file_name: "Pages-SUB-Convert"     # 生成的配置文件名称
sub_update_time: 6                 # 订阅更新时间（小时）

# Telegram通知配置（可选）
tg_bot_token: ""                   # Telegram Bot Token
tg_chat_id: ""                     # Telegram Chat ID
tg_notify_level: 1                 # 通知级别：1=所有请求，0=仅异常访问

# 订阅转换配置
subconverter: "apiurl.v1.mk"       # 订阅转换后端地址
sub_config: "https://raw.githubusercontent.com/cmliu/ACL4SSR/main/Clash/config/ACL4SSR_Online_MultiCountry.ini"  # 订阅转换配置文件
```

## API 使用说明

### 1. 获取订阅内容

```bash
http://your-domain:8080/sub?token=your_token
```

### 2. 管理订阅链接

添加订阅：
```bash
curl -X POST "http://your-domain:8080/api/subscribe?token=your_token" \
     -H "Content-Type: application/json" \
     -d '{"url":"https://example.com/sub"}'
```

删除订阅：
```bash
curl -X DELETE "http://your-domain:8080/api/subscribe?token=your_token" \
     -H "Content-Type: application/json" \
     -d '{"url":"https://example.com/sub"}'
```

查看所有订阅：
```bash
curl "http://your-domain:8080/api/subscribe?token=your_token"
```

## 编译说明

1. 安装 Go 1.21 或更高版本
2. 克隆仓库：
```bash
git clone https://github.com/yourusername/sublinks.git
cd sublinks
```

3. 编译：
```bash
# Linux/macOS
chmod +x build.sh
./build.sh

# Windows
go build -o sublinks.exe cmd/main.go
```

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License 