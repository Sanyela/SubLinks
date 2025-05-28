#!/bin/bash

# 版本号
VERSION="v1.0.0"

# 创建构建目录
mkdir -p build

# 编译 Linux 版本
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o build/sublinks-linux-amd64 cmd/main.go
GOOS=linux GOARCH=arm64 go build -o build/sublinks-linux-arm64 cmd/main.go

# 编译 Windows 版本
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o build/sublinks-windows-amd64.exe cmd/main.go
GOOS=windows GOARCH=386 go build -o build/sublinks-windows-386.exe cmd/main.go

# 复制配置文件模板
cp config.yaml.example build/

# 创建版本压缩包
cd build
for file in sublinks-*; do
    if [[ $file == *.exe ]]; then
        zip "${file%.exe}-${VERSION}.zip" "$file" config.yaml.example
    else
        tar -czf "${file}-${VERSION}.tar.gz" "$file" config.yaml.example
    fi
done

echo "Build complete!" 