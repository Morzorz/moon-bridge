# Moon Bridge GUI

Moon Bridge 管理后台前端，基于 **Vue 3 + Vite** 构建，采用暗色主题设计系统。

## 技术栈

| 层级 | 技术 |
|------|------|
| 框架 | Vue 3 (Composition API + `<script setup>`) |
| 路由 | Vue Router 4 (Hash 模式) |
| 构建 | Vite 6 |
| HTTP | Fetch API (原生) |
| 字体 | Inter (UI) + JetBrains Mono (代码) |
| 图表 | 预留 Recharts / ECharts 集成位 |
| 桌面 | 预留 Wails v2/v3 迁移接口 |

## 快速开始

```bash
# 安装依赖
cd gui-design
npm install

# 开发模式 (需要 Moon Bridge 后端已在 38440 端口运行)
npm run dev
# → 访问 http://localhost:38441

# 构建
npm run build
# → dist/ 目录产出静态文件
```

开发模式下 Vite proxy 将 `/api/*` 转发到 `http://127.0.0.1:38440`。

## 项目结构

```
gui-design/
├── index.html              # 入口 HTML
├── package.json
├── vite.config.js           # Vite 配置 (含 API proxy)
├── README.md
└── src/
    ├── main.js              # Vue 应用引导
    ├── App.vue              # 根组件 (侧边栏 + Toast 系统)
    ├── assets/
    │   └── style.css        # 全局设计系统 (深色主题)
    ├── api/
    │   └── index.js         # Moon Bridge REST API 客户端
    ├── router/
    │   └── index.js         # 路由配置 (10 个页面)
    ├── views/
    │   ├── Dashboard.vue    # 仪表盘
    │   ├── Providers.vue    # Provider 管理
    │   ├── Models.vue       # 模型 + Offers 管理
    │   ├── Routes.vue       # 路由管理
    │   ├── Settings.vue     # 设置 (Defaults/WebSearch/Extensions)
    │   ├── Config.vue       # 配置导入/导出/验证
    │   ├── Changes.vue      # 变更管理 (暂存/应用/丢弃)
    │   ├── Sessions.vue     # 会话列表
    │   ├── Stats.vue        # 统计分析
    │   └── Logs.vue         # 日志查看
    └── composables/
        ├── useApi.js        # 通用 API 调用组合式函数
        └── useToast.js      # Toast 通知注入
```

## Wails 集成指南

> Moon Bridge 后端是 Go 项目，天然适合用 [Wails](https://wails.io) 打包为独立桌面应用。

### 集成步骤

1. **安装 Wails CLI**
   ```bash
   go install github.com/wailsapp/wails/v3/cmd/wails@latest
   ```

2. **在项目根目录初始化 Wails**
   ```bash
   cd ..
   wails init -n moonbridge-app
   ```

3. **替换前端目录**  
   将 `wails init` 生成的前端目录替换为本项目的 `gui-design/`：
   ```bash
   rm -rf moonbridge-app/frontend
   ln -s ../gui-design moonbridge-app/frontend
   ```
   或直接拷贝：
   ```bash
   cp -r gui-design moonbridge-app/frontend
   ```

4. **为 Moon Bridge 创建 Wails 绑定**  
   在 `cmd/wails/` 下创建 Go 入口，调用 `internal/service/app.RunServer`：
   ```go
   package main

   import (
       "context"
     "moonbridge/internal/config"
     "moonbridge/internal/service/app"
     "github.com/wailsapp/wails/v3/pkg/application"
   )

   func main() {
     cfg, _ := config.Load("config.yml")
     app.New(application.Options{
       Assets: application.AssetFileServerFS(frontend.DistFS),
     })

     go app.RunServer(context.Background(), *cfg, os.Stderr)
     app.Run()
   }
   ```

5. **构建桌面应用**
   ```bash
   wails build
   ```

### 需要修改的地方

| 位置 | 开发模式 | Wails 模式 |
|------|----------|------------|
| `src/api/index.js` | `BASE_URL = '/api/v1'` (Vite proxy) | `BASE_URL = 'http://127.0.0.1:38440/api/v1'` 或 Wails runtime 接管 |
| `src/router/index.js` | Hash 路由 | Hash 路由 (与 WebView 兼容) |
| `vite.config.js` | proxy → `localhost:38440` | 移除 proxy，或通过环境变量控制 |

### 环境变量切换

```bash
# Wails 模式下设置
VITE_API_BASE=http://127.0.0.1:38440/api/v1
```

Wails 提供 `wails.runtime` JS 对象，可用于替代 fetch 直接调用 Go 方法（可选）。

## 页面列表

| # | 页面 | 路由 | 主要 API 端点 |
|---|------|------|--------------|
| 1 | 仪表盘 | `/` | `status`, `stats/summary`, `sessions` |
| 2 | Providers | `/providers` | `providers/*` (CRUD + test) |
| 3 | 模型管理 | `/models` | `models/*`, `providers/*/offers` |
| 4 | 路由管理 | `/routes` | `routes/*` |
| 5 | 设置 | `/settings` | `defaults`, `web-search`, `extensions` |
| 6 | 配置管理 | `/config` | `config/effective`, `config/export/import/validate` |
| 7 | 变更管理 | `/changes` | `changes/*` (list/apply/discard) |
| 8 | 会话 | `/sessions` | `sessions` |
| 9 | 统计分析 | `/stats` | `stats`, `stats/summary` |
| 10 | 日志 | `/logs` | `logs` |
