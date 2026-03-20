# XScan Web - XSS扫描管理平台

一个基于 Web 的 XSS 自动化扫描管理平台，将 xscan 工具封装为 Web 服务，支持通过浏览器一键完成「子域名收集 → 存活探测 → XSS扫描 → 报告查看」的完整流程。

## 架构

```
┌─────────────────┐       HTTP API        ┌──────────────────────────────┐
│   前端 (Vue3)     │ ◄──────────────────► │    后端 API (Go/Gin)          │
│   Element Plus   │                      │    SQLite + Task Queue       │
└─────────────────┘                      └──────────┬───────────────────┘
                                                     │
                                          ┌──────────▼───────────────────┐
                                          │       任务调度引擎              │
                                          │  1. subfinder 子域名收集       │
                                          │  2. httpx 存活探测            │
                                          │  3. xscan spider XSS扫描     │
                                          │  4. 解析报告 & 入库             │
                                          └──────────────────────────────┘
```

## 功能特性

- 🎯 **一键扫描**: 输入根域名，自动完成子域名收集 → 存活探测 → XSS扫描
- 📊 **任务管理**: 创建、查看、删除扫描任务，实时跟踪扫描进度
- 📋 **结果展示**: 查看子域名列表、存活URL、XSS漏洞详情
- 📄 **报告查看**: 支持 Markdown 格式的 XSS 报告在线查看
- 🔐 **Token 认证**: 简单 Token 保护，防止未授权访问
- 📱 **响应式界面**: 适配桌面和移动端浏览器

## 前置要求

VPS 上需要安装：
- **Go** 1.21+ (编译后端)
- **Node.js** 18+ (构建前端)
- **xscan** (XSS扫描工具)
- **subfinder** (子域名收集，可选)
- **httpx** (存活探测，可选)

## 快速部署

### 方式一：一键部署脚本（推荐）

```bash
# 1. 上传项目到 VPS
scp -r ./* user@your-vps:/opt/xscan-web/

# 2. SSH 到 VPS
ssh user@your-vps

# 3. 运行部署脚本
cd /opt/xscan-web
chmod +x deploy.sh
./deploy.sh

# 4. 修改配置
vim config.json   # 修改 auth_token

# 5. 启动服务
./xscan-web-server
```

### 方式二：Docker Compose

```bash
# 1. 准备 xscan
mkdir -p xscan tools
cp /path/to/xscan ./xscan/
cp /path/to/config.yaml ./xscan/
cp -r /path/to/dict ./xscan/

# 2. 准备工具
cp $(which subfinder) ./tools/
cp $(which httpx) ./tools/

# 3. 修改配置
vim backend/config.json   # 修改 auth_token

# 4. 启动
docker-compose up -d
```

### 方式三：手动部署

```bash
# 1. 构建后端
cd backend
go mod tidy
CGO_ENABLED=1 go build -o ../xscan-web-server .
cd ..

# 2. 构建前端
cd frontend
npm install
npm run build
cd ..

# 3. 部署前端静态文件
cp -r frontend/dist static

# 4. 准备目录
mkdir -p data results tools xscan

# 5. 放置 xscan
cp -r xscan_3.6.5_linux_amd64/* ./xscan/
chmod +x ./xscan/xscan

# 6. 配置
cp backend/config.json config.json
vim config.json

# 7. 启动
./xscan-web-server
```

## 配置说明

`config.json`:

```json
{
  "port": 8080,              // 后端监听端口
  "db_path": "./data/xscan.db",  // SQLite 数据库路径
  "xscan_path": "./xscan/xscan", // xscan 可执行文件路径
  "tools_dir": "./tools",        // 工具目录 (subfinder, httpx)
  "results_dir": "./results",    // 扫描结果目录
  "max_concurrent": 2,           // 最大并发任务数
  "auth_token": "your-secret"    // 访问 Token (留空则不验证)
}
```

## 扫描流程

```
输入: example.com
    │
    ▼
[Step 1] subfinder -d example.com → 收集子域名
    │
    ▼
[Step 2] httpx 探测存活 → 过滤可访问的 URL
    │
    ▼
[Step 3] xscan spider -u <url> → 逐个 URL 进行 XSS 扫描
    │
    ▼
[Step 4] 解析 *_xss.md 报告 → 存入数据库
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/tasks` | 创建扫描任务 |
| GET | `/api/tasks` | 获取任务列表 |
| GET | `/api/tasks/:id` | 获取任务详情 |
| DELETE | `/api/tasks/:id` | 删除任务 |
| GET | `/api/tasks/:id/report` | 获取XSS报告 |
| GET | `/health` | 健康检查 (无需认证) |

请求头需携带: `Authorization: Bearer <your-token>`

## 目录结构

```
xscan-web/
├── backend/                 # Go 后端
│   ├── main.go             # 入口
│   ├── go.mod
│   ├── config.json         # 默认配置
│   ├── Dockerfile
│   └── internal/
│       ├── config/         # 配置管理
│       ├── database/       # SQLite 数据库
│       ├── handler/        # API 处理器
│       ├── models/         # 数据模型
│       └── scanner/        # 扫描调度引擎
├── frontend/               # Vue3 前端
│   ├── package.json
│   ├── vite.config.js
│   ├── index.html
│   ├── Dockerfile
│   ├── nginx.conf
│   └── src/
│       ├── main.js
│       ├── App.vue
│       ├── api/            # API 封装
│       ├── router/         # 路由
│       └── views/          # 页面组件
│           ├── TaskList.vue    # 任务列表
│           └── TaskDetail.vue  # 任务详情
├── docker-compose.yml
├── deploy.sh               # 部署脚本
└── README.md
```

## 使用 systemd 管理（推荐）

```bash
# 创建 service 文件
sudo cat > /etc/systemd/system/xscan-web.service << 'EOF'
[Unit]
Description=XScan Web Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xscan-web
ExecStart=/opt/xscan-web/xscan-web-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 启用并启动
sudo systemctl daemon-reload
sudo systemctl enable xscan-web
sudo systemctl start xscan-web

# 查看状态
sudo systemctl status xscan-web

# 查看日志
sudo journalctl -u xscan-web -f
```

## 安全建议

1. 修改默认 `auth_token`，使用强密码
2. 使用 nginx 反向代理并配置 HTTPS
3. 限制 VPS 防火墙端口访问
4. 定期清理旧的扫描结果

## License

MIT
