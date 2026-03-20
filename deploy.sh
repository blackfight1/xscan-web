#!/bin/bash
# XScan Web 一键部署脚本
# 使用方法: chmod +x deploy.sh && ./deploy.sh

set -e

echo "=========================================="
echo "  XScan Web 部署脚本"
echo "=========================================="

# 创建必要目录
echo "[1/6] 创建目录结构..."
mkdir -p data results tools xscan

# 检查 xscan 是否存在
if [ ! -f "./xscan/xscan" ]; then
    echo "[!] 请将 xscan 二进制文件放到 ./xscan/ 目录下"
    echo "    例如: cp /path/to/xscan ./xscan/xscan"
    echo "    同时复制 config.yaml 和 dict 目录"

    # 如果 xscan_3.6.5_linux_amd64 目录存在，自动复制
    if [ -d "./xscan_3.6.5_linux_amd64" ]; then
        echo "[*] 检测到 xscan_3.6.5_linux_amd64 目录，自动复制..."
        cp -r ./xscan_3.6.5_linux_amd64/* ./xscan/
        chmod +x ./xscan/xscan
        echo "[✓] xscan 已复制到 ./xscan/"
    fi
fi

# 检查 subfinder
echo "[2/6] 检查工具..."
if ! command -v subfinder &> /dev/null; then
    echo "[*] 安装 subfinder..."
    if command -v go &> /dev/null; then
        go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest
        cp $(go env GOPATH)/bin/subfinder ./tools/ 2>/dev/null || true
    else
        echo "[!] subfinder 未安装，请手动安装: https://github.com/projectdiscovery/subfinder"
    fi
else
    echo "[✓] subfinder 已安装"
    cp $(which subfinder) ./tools/ 2>/dev/null || true
fi

# 检查 httpx
if ! command -v httpx &> /dev/null; then
    echo "[*] 安装 httpx..."
    if command -v go &> /dev/null; then
        go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest
        cp $(go env GOPATH)/bin/httpx ./tools/ 2>/dev/null || true
    else
        echo "[!] httpx 未安装，请手动安装: https://github.com/projectdiscovery/httpx"
    fi
else
    echo "[✓] httpx 已安装"
    cp $(which httpx) ./tools/ 2>/dev/null || true
fi

# 构建后端
echo "[3/6] 构建后端..."
cd backend
if [ ! -f "go.sum" ]; then
    go mod tidy
fi
CGO_ENABLED=1 go build -o ../xscan-web-server .
cd ..
echo "[✓] 后端构建完成"

# 构建前端
echo "[4/6] 构建前端..."
cd frontend
npm install
npm run build
cd ..

# 复制前端到后端 static 目录
echo "[5/6] 部署前端静态文件..."
rm -rf static
cp -r frontend/dist static
echo "[✓] 前端部署完成"

# 生成配置
echo "[6/6] 配置检查..."
if [ ! -f "config.json" ]; then
    cp backend/config.json config.json
    echo "[!] 已生成默认 config.json，请修改 auth_token"
fi

echo ""
echo "=========================================="
echo "  部署完成！"
echo "=========================================="
echo ""
echo "目录结构:"
echo "  ./xscan-web-server   - 后端可执行文件"
echo "  ./config.json        - 配置文件"
echo "  ./static/            - 前端静态文件"
echo "  ./xscan/             - xscan 工具"
echo "  ./tools/             - subfinder, httpx 等"
echo "  ./data/              - SQLite 数据库"
echo "  ./results/           - 扫描结果"
echo ""
echo "启动方式:"
echo "  ./xscan-web-server"
echo ""
echo "或使用 Docker Compose:"
echo "  docker-compose up -d"
echo ""
echo "注意: 请修改 config.json 中的 auth_token"
echo "=========================================="
