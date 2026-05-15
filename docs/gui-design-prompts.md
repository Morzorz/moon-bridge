# Moon Bridge GUI 管理后台 — 页面设计提示词集

> 本文档为 LLM 协议转换与模型路由代理服务器 **Moon Bridge** 的 GUI 管理后台页面设计提示词集。
> 每个页面提示词均可独立投喂给 AI 设计/编码工具（v0、Bolt、Lovable、Cursor 等）。
> 后台管理 API 基路径均为 `/api/v1`。

---

## 目录

1. [仪表盘 Dashboard](#1-仪表盘-dashboard)
2. [Provider 管理页](#2-provider-管理页)
3. [模型管理页](#3-模型管理页)
4. [路由管理页](#4-路由管理页)
5. [设置页](#5-设置页)
6. [配置导入/导出页](#6-配置导入导出页)
7. [变更管理页](#7-变更管理页)
8. [会话管理页](#8-会话管理页)
9. [统计与分析页](#9-统计与分析页)
10. [系统日志页](#10-系统日志页)
11. [全局设计系统](#11-全局设计系统)

---

## 1. 仪表盘 Dashboard

```
为一个 LLM 代理服务器「Moon Bridge」设计一个仪表盘 Dashboard 页面。

## 页面用途
作为管理后台首页，展示服务器整体运行状态和关键指标。

## 数据来源
- GET /api/v1/status 返回: { version, mode, provider_count, route_count, addr, uptime, timestamp }
- GET /api/v1/stats/summary 返回: { requests, input_tokens, output_tokens, cache_hit_rate, total_cost, duration }
- GET /api/v1/status/providers 返回: [{ key, protocol, base_url, offer_count, health_status }]

## 页面布局
采用卡片网格布局（响应式，大屏 3 列，中屏 2 列，小屏 1 列）：

### 顶部统计卡片行（4 个卡片）
1. **运行模式** — 显示当前 mode（Transform/CaptureAnthropic/CaptureResponse），用不同颜色标签区分
2. **Provider 数量** — 数字 + "个 Provider" + 健康/总数比
3. **路由数量** — 数字 + "条路由规则"
4. **监听地址** — 显示 addr

### 用量概览卡片行（4 个卡片）
1. **总请求数** — 大数字，带请求趋势箭头
2. **Token 消耗** — 输入/输出分别显示，格式化为 K/M
3. **缓存命中率** — 百分比 + 环形进度条
4. **累计费用** — ¥ 金额（RMB），保留 4 位小数

### 下方两栏
- **左栏 (60%)**：Provider 健康状态列表（表格），列：名称、协议、Base URL、Offer 数、健康状态（绿/黄/红 圆点）
- **右栏 (40%)**：最近会话列表（GET /api/v1/sessions），显示会话标识（脱敏）、模型名、最后活跃时间

### 样式要求
- 深色主题优先（暗色背景），科技蓝为主色调
- 卡片带微弱阴影和圆角
- 数字使用等宽字体
- 加载态：骨架屏
- 错误态：每个卡片独立处理，失败显示红色错误提示
- 自动刷新：每 10 秒轮询一次，刷新时不闪烁
```

---

## 2. Provider 管理页

```
为 LLM 代理服务器「Moon Bridge」设计一个 Provider 管理页面。

## 页面用途
管理上游 LLM Provider（如 OpenAI、Anthropic、DeepSeek、Kimi），支持增删改查和连通性测试。

## 数据来源
- GET /api/v1/providers — 列表，支持分页 { data: [{ key, protocol, offer_count, base_url, health_status }], total, limit, offset }
- GET /api/v1/providers/{key} — 单个详情
- PUT /api/v1/providers/{key} — 创建/更新 (body: { protocol, base_url, api_key, version, user_agent, headers? })
- PATCH /api/v1/providers/{key} — 部分更新
- DELETE /api/v1/providers/{key} — 删除
- POST /api/v1/providers/{key}/test — 连通性测试 (返回: { success, provider, base_url, duration, error?, timestamp })

注意：所有修改操作不会立即生效，而是暂存为 pending change（返回 { change_id, status: "pending" }），需要在「变更管理」页统一 apply。

## 页面布局

### 顶部工具栏
- 搜索框（按 provider key 模糊搜索）
- 协议筛选下拉框（anthropic / openai-response / google-genai / openai-chat）
- 「添加 Provider」按钮（主色调，右侧）

### Provider 卡片列表
每个 Provider 展示为一张卡片（非表格），卡片内容：
- 头部：Provider Key（粗体）+ 协议标签（彩色徽章）+ 健康状态圆点
- 主体：Base URL（等宽字体，可复制）、API Key（脱敏显示 `****abcd`，带显示/隐藏切换）
- 底部：Offer 数量 + 「测试连接」按钮 + 操作菜单（编辑/删除）

### 添加/编辑弹窗（Modal）
表单字段：
- **Key**（必填，新建时输入，编辑时只读）
- **Protocol**（下拉选择：anthropic / openai-response / google-genai / openai-chat）
- **Base URL**（URL 输入框）
- **API Key**（密码输入框，带显示/隐藏切换）
- **Version**（选填，如 "2023-06-01"）
- **User Agent**（选填）
- 保存按钮（提交 PUT）+ 取消按钮

### 连通性测试
点击「测试连接」按钮后：
- 按钮变为加载旋转图标 + "测试中..."
- 结果以 Toast 或内联提示展示：成功显示绿色 + 耗时，失败显示红色 + 错误信息

### 样式要求
- 卡片悬停效果（微微上浮 + 阴影加深）
- 协议标签用不同颜色：anthropic=紫色，openai-response=绿色，google-genai=蓝色，openai-chat=橙色
- 删除操作需二次确认弹窗
```

---

## 3. 模型管理页

```
为 LLM 代理服务器「Moon Bridge」设计一个模型管理页面。

## 页面用途
管理模型定义（slug、显示名、上下文窗口大小），以及各 Provider 对模型的定价 Offer。

## 数据来源
- GET /api/v1/models — 列表 { data: [{ slug, display_name, context_window, providers }], total, limit, offset }
- GET /api/v1/models/{slug} — 单个详情
- PUT /api/v1/models/{slug} — 创建/更新 (body: { display_name, context_window })
- DELETE /api/v1/models/{slug} — 删除（如果仍被 provider offer 引用则返回 409 拒绝）
- POST /api/v1/providers/{key}/offers — 为 provider 创建 offer (body: { model, upstream_name, priority, input_price, output_price, cache_write, cache_read })
- PATCH /api/v1/providers/{key}/offers/{model} — 更新 offer 定价
- DELETE /api/v1/providers/{key}/offers/{model} — 删除 offer

## 页面布局

### 顶部工具栏
- 搜索框（按 slug 模糊搜索）
- 「添加模型」按钮

### 模型表格
列：Slug、显示名称、Context Window、关联 Provider 数、操作
- 「关联 Provider」列显示 provider 标签列表（可点击跳转到 provider 详情）
- 操作：编辑、删除、展开查看 Offers

### 展开行（Offers 详情）
点击展开后显示该模型在各 Provider 下的定价：
- 表格列：Provider、上游模型名、优先级、Input Price、Output Price、Cache Write、Cache Read
- 定价单位：¥/M tokens
- 支持在展开行内直接修改定价（PATCH）或删除 offer
- 「添加 Offer」按钮：弹出小表单，选择 Provider + 填写定价

### 添加/编辑模型弹窗
- Slug（必填，如 "deepseek-v4-pro"）
- Display Name（选填，如 "DeepSeek V4 Pro"）
- Context Window（数字，如 128000）

### 样式要求
- 表格支持排序（点击列头）
- Context Window 数字格式化为 K（如 128000 → 128K）
- 定价以 ¥ 显示，保留 2 位小数
- Offer 为空时显示 "暂无 Provider 提供此模型" 空状态
```

---

## 4. 路由管理页

```
为 LLM 代理服务器「Moon Bridge」设计一个路由管理页面。

## 页面用途
管理模型别名路由规则，将客户端请求的模型别名映射到具体 Provider 的上游模型。

## 数据来源
- GET /api/v1/routes — 列表 { data: [{ alias, model, provider, display_name }], total, limit, offset }
- GET /api/v1/routes/{alias} — 单个详情，含 context_window
- PUT /api/v1/routes/{alias} — 创建/更新 (body: { model, provider, display_name, context_window })
- DELETE /api/v1/routes/{alias} — 删除

路由说明：alias 是客户端使用的友好名称，model 是上游模型 slug，provider 是 provider key。
例如：alias="moonbridge" → model="deepseek-v4-pro" provider="deepseek"

## 页面布局

### 顶部工具栏
- 「添加路由」按钮

### 路由表格
列：Alias、目标模型、Provider、显示名称、Context Window、操作
- Alias 用等宽字体 + 可复制
- 目标模型和 Provider 用 `→` 箭头连接（如 `deepseek-v4-pro → deepseek`）
- 操作：编辑、删除

### 添加/编辑路由表单（可用右侧抽屉 Drawer 或 Modal）
- **Alias**（必填，如 "moonbridge"、"gpt-image"）
- **Model**（必填，下拉选择已有模型 slug）
- **Provider**（必填，下拉选择已有 provider key）
- **Display Name**（选填）
- **Context Window**（选填数字，不填则继承模型定义）

### 辅助功能
- Model 和 Provider 下拉框旁有「新建」快捷链接，点击跳转到对应管理页
- 列表为空时显示引导性空状态："还没有路由规则，创建第一条路由将客户端模型名映射到上游 Provider"

### 样式要求
- 使用 `→` 箭头和不同颜色区分 alias（青色）和 provider（蓝色）
- 表格行悬停高亮
```

---

## 5. 设置页

```
为 LLM 代理服务器「Moon Bridge」设计一个设置页面。

## 页面用途
集中管理默认参数、Web Search 配置和扩展（Extension）开关。

## 数据来源
- GET /api/v1/defaults — { model, max_tokens, system_prompt }
- PUT /api/v1/defaults — (body: { model, max_tokens, system_prompt }) → { change_id, status: "pending" }
- GET /api/v1/web-search — { support, max_uses, tavily_api_key, firecrawl_api_key, search_max_rounds }
- PUT /api/v1/web-search — (body: { support, max_uses, tavily_api_key, firecrawl_api_key, search_max_rounds })
- GET /api/v1/extensions — ["deepseek_v4", "visual", "kimi_workaround", "metrics", ...]
- GET /api/v1/extensions/{name} — 单个扩展配置详情
- PUT /api/v1/extensions/{name} — 更新扩展配置

## 页面布局
采用左侧 Tab 导航 + 右侧内容区的布局：

### Tab 1: 默认参数 (Defaults)
- **默认模型**：下拉选择框（从 /api/v1/models 获取列表）
- **默认 Max Tokens**：数字输入框
- **默认 System Prompt**：多行文本域（等宽字体，4 行高）
- 「保存」按钮

提示文字："当客户端请求未指定 model/max_tokens/system_prompt 时使用这些默认值"

### Tab 2: Web Search
- **搜索支持模式**：下拉选择（disabled / enabled / auto）
  - disabled: 完全禁用
  - enabled: 客户端可主动调用
  - auto: 自动注入搜索工具
- **最大使用次数 (max_uses)**：数字输入
- **最大搜索轮次 (search_max_rounds)**：数字输入
- **Tavily API Key**：密码输入框（脱敏显示，带显示/隐藏）
- **Firecrawl API Key**：密码输入框（脱敏显示，带显示/隐藏）
- 「保存」按钮

### Tab 3: 扩展管理 (Extensions)
- 扩展列表（每个扩展一个卡片/折叠面板）：
  - **deepseek_v4**：Reinforce Instructions 开关、自定义 Reinforce Prompt 文本框
  - **visual**：Provider 选择、Model 选择、Max Rounds、Max Tokens
  - **kimi_workaround**：Max Tool Rounds、Convergence Margin
  - **metrics**：Default Limit、Max Limit
- 每个扩展独立「保存」按钮

### 通用要求
- 所有保存操作生成 pending change，显示 "变更已暂存，需在变更管理页应用"
- API Key 类字段默认脱敏（显示 `****` + 最后 4 位），带切换按钮
```

---

## 6. 配置导入/导出页

```
为 LLM 代理服务器「Moon Bridge」设计一个配置管理页面。

## 页面用途
查看当前生效配置、导出 YAML、导入新配置、验证配置合法性。

## 数据来源
- GET /api/v1/config/effective — 当前生效的完整配置 JSON
- GET /api/v1/config/export — 导出为 YAML 格式
- POST /api/v1/config/import — 导入 YAML 配置 (body: { config: "<yaml string>" }) → { change_id, status }
- POST /api/v1/config/validate — 验证配置 (body: { config: "<yaml string>" }) → { valid, errors?: [{ path, message }] }

## 页面布局

### 三栏式水平布局

#### 左栏（30%）— 操作面板
- **当前配置信息**：版本号、Provider 数、Route 数、加载时间
- 「导出 YAML」按钮 — 点击下载 config.yml 文件
- 「验证配置」按钮 — 展开文本域粘贴 YAML 后验证
- 「导入配置」按钮 — 展开文本域粘贴 YAML 后导入

#### 右栏（70%）— YAML 编辑器
- 使用代码编辑器组件（等宽字体、语法高亮、行号）
- 两个模式切换：
  - **查看模式**：只读，显示当前生效配置的 YAML
  - **编辑模式**：可编辑，用于粘贴新配置进行验证或导入

### 验证流程
1. 用户在编辑器中粘贴/编辑 YAML
2. 点击「验证配置」
3. 结果显示在编辑器下方：
   - ✅ 通过：绿色横幅 "配置格式正确，可以导入"
   - ❌ 失败：红色错误列表，每个错误显示 path + message，编辑器对应行高亮

### 导入流程
1. 点击「导入配置」→ 弹出确认对话框
2. 确认后提交 → 返回 change_id
3. 显示 "配置已暂存为待处理变更" + 跳转链接到变更管理页

### 样式要求
- YAML 编辑器使用深色背景 + 语法高亮（key 用蓝色，value 用绿色，注释用灰色）
- 大段文本区域高度至少 400px
- 导入/导出按钮带图标（↓ 下载 / ↑ 上传）
```

---

## 7. 变更管理页

```
为 LLM 代理服务器「Moon Bridge」设计一个变更管理（Pending Changes）页面。

## 页面用途
查看所有暂存的配置变更，统一应用或丢弃。Moon Bridge 的修改操作（增删改 Provider/Model/Route/Setting）不会立即生效，而是暂存为 pending change，需要在此页统一 Apply。

## 数据来源
- GET /api/v1/changes — 变更列表 [{ id, action, resource, target_key, before?, after, created_at }]
- POST /api/v1/changes/apply — 应用所有变更 → { status: "success", message: "变更已应用生效" }
- POST /api/v1/changes/discard — 丢弃所有变更 → { status: "success", message: "变更已丢弃" }

## 页面布局

### 顶部摘要栏
- 待处理变更数量（大数字 + 徽章）
- 按 resource 分类统计（Provider: N, Model: N, Route: N, Setting: N）
- 两个主操作按钮：
  - 「应用全部变更」(Apply) — 绿色主按钮
  - 「丢弃全部变更」(Discard) — 红色次按钮（需二次确认）

### 变更列表
每项变更展示为一条时间线卡片：

- **左侧时间轴**：竖线 + 圆点（create=绿色圆点, update=蓝色圆点, delete=红色圆点）
- **卡片内容**：
  - 标题：`{action} {resource} "{target_key}"`
  - action 标签：CREATE（绿）/ UPDATE（蓝）/ DELETE（红）徽章
  - 变更对比（diff 视图）：
    - CREATE：显示 after 的完整 JSON（绿色背景）
    - UPDATE：左右并排对比 before/after（红色删 + 绿色增）
    - DELETE：显示 before 的完整 JSON，标注 "将被删除"（红色背景淡色）
  - 创建时间

### 空状态
当无待处理变更时：
- 大图标（✓）
- "没有待处理的变更"
- "在 Provider / Model / Route / 设置 页做的修改会自动暂存到这里"
- 引导链接跳转到各管理页

### 操作确认
- Apply 按钮点击后弹出确认框："确定要应用所有 N 项变更吗？应用后立即生效。"
- Discard 按钮点击后弹出警告框（红色）："确定要丢弃所有 N 项变更吗？此操作不可撤销。"

### 样式要求
- 时间轴线条用细腻的灰色，圆点用对应操作颜色
- JSON diff 使用等宽字体，增/删行有明显的背景色区分
- Apply 成功/失败用 Toast 通知
```

---

## 8. 会话管理页

```
为 LLM 代理服务器「Moon Bridge」设计一个会话管理页面。

## 页面用途
查看所有活跃的客户端会话列表。

## 数据来源
- GET /api/v1/sessions — [{ key, model, created_at, last_used }]
  - key 已脱敏（如 `sess_****a1b2`）
  - model 为客户端使用的模型名

## 页面布局

### 顶部统计行
- 活跃会话总数
- 使用的不同模型数（去重计数）

### 会话表格
列：会话标识（脱敏）、使用模型、创建时间、最后活跃时间、存活时长

- 「使用模型」列显示为彩色标签
- 「存活时长」= 当前时间 - 创建时间（如 "2h 15m"）
- 「最后活跃时间」显示相对时间（如 "3 分钟前"），悬停显示绝对时间
- 表格支持按「最后活跃时间」降序排列（默认）

### 自动刷新
- 每 15 秒自动刷新会话列表
- 页面顶部显示 "自动刷新中" 指示器（绿色脉冲圆点），可手动暂停

### 空状态
- "当前没有活跃会话"
- 图标：空列表插画

### 样式要求
- 表格行 hover 高亮
- 模型标签用不同颜色区分不同模型
- 相对时间自动更新（用 `timeago` 风格显示）
```

---

## 9. 统计与分析页

```
为 LLM 代理服务器「Moon Bridge」设计一个统计与分析页面。

## 页面用途
可视化展示 LLM 代理的用量统计数据，包括请求量、Token 消耗、费用和缓存命中率。

## 数据来源
- GET /api/v1/stats — 完整统计 JSON
- GET /api/v1/stats/summary — { requests, input_tokens, output_tokens, cache_hit_rate, total_cost, duration }

实际 stats 数据结构（来自后端）包含：
- totalRequests, totalInputTokens, totalOutputTokens
- totalCacheCreation, totalCacheRead
- totalCost (RMB)
- byModel: { modelName: { requests, inputTokens, outputTokens, cost, cacheCreation, cacheRead } }

## 页面布局

### 顶部概览行（5 个统计卡片）
1. **总请求数** — 大数字 + 运行时长
2. **Input Tokens** — 格式化数字 + "输入"标签
3. **Output Tokens** — 格式化数字 + "输出"标签
4. **缓存命中率** — 百分比 + 环形进度图
5. **累计费用** — ¥XX.XXXX + RMB 标签

### 中部图表区（2 列）

#### 左列：Token 消耗趋势（柱状图/折线图）
- 如果有时序数据可按时间展示
- 否则按模型展示：X 轴=模型名，Y 轴=Token 数，堆叠 input/output

#### 右列：费用分布（饼图/环形图）
- 按模型分组的费用占比
- 中心显示总费用
- 图例显示模型名 + 金额 + 百分比

### 底部：按模型明细表
表格列：模型名、请求数、Input Tokens、Output Tokens、缓存写入、缓存读取、费用(¥)、平均费用/请求
- 支持按任意列排序
- 最后一行显示合计
- Token 数字格式化（1234567 → 1.23M）
- 费用保留 6 位小数

### 空状态
- "暂无统计数据" + "发送请求后统计数据将自动汇总到这里"

### 样式要求
- 图表使用 Recharts / ECharts 等库
- 配色方案：input=蓝色，output=绿色，cache=紫色，cost=金色
- 数字使用等宽字体，大数字加粗
```

---

## 10. 系统日志页

```
为 LLM 代理服务器「Moon Bridge」设计一个系统日志查看页面。

## 页面用途
查看服务器运行日志（当前后端为 placeholder，返回空数组，需按未来实现设计）。

## 数据来源
- GET /api/v1/logs — 当前返回 []（空数组）
- 未来预期返回: [{ timestamp, level, message, source? }]

## 页面布局

### 顶部控制栏
- 日志级别筛选：ALL / DEBUG / INFO / WARN / ERROR（标签式切换）
- 搜索框：按消息内容模糊搜索
- 自动滚动开关（默认开启）
- 「清空」按钮
- 「暂停/恢复」按钮（暂停实时更新）

### 日志列表区
- 虚拟滚动列表（日志量大时保持性能）
- 每条日志一行：
  - **时间戳**（灰色，等宽字体）`2024-01-15 14:32:05.123`
  - **级别标签**（彩色徽章）：DEBUG=灰色, INFO=蓝色, WARN=橙色, ERROR=红色
  - **消息内容**（等宽字体，自动换行）
- 点击某行日志可展开查看详细信息（JSON 格式化显示）

### 底栏状态
- 显示总日志条数
- 实时连接状态指示器（绿色圆点 + "实时" 或 红色 + "已断开"）

### 空状态（当前后端返回空数组时）
- 居中显示："日志功能即将上线"
- 副标题："未来将支持实时滚动查看服务器运行日志"
- 图标：终端图标

### 样式要求
- 深色终端风格背景（#1a1a2e 或类似）
- 等宽字体全局使用
- ERROR 级别行有淡红色背景高亮
- 日志区占满剩余高度
```

---

## 11. 全局设计系统

```
为 Moon Bridge 管理后台定义全局设计系统。

## 品牌
- 产品名：Moon Bridge
- 标语：LLM 协议转换与模型路由代理
- 图标/Logo：桥 + 月亮的组合意象

## 配色方案（暗色主题优先）
- 主背景：#0f0f1a（深蓝黑）
- 卡片背景：#1a1a2e
- 主色调：#4fc3f7（科技蓝）
- 成功色：#66bb6a（绿）
- 警告色：#ffa726（橙）
- 错误色：#ef5350（红）
- 文字主色：#e0e0e0
- 文字次色：#9e9e9e

## 排版
- 标题：系统字体（思源黑体 / Inter），粗体
- 正文：系统字体，常规
- 代码/数字：JetBrains Mono / Fira Code，等宽
- 数字格式化：>1000 用 K，>1000000 用 M

## 布局
- 侧边栏导航（左侧 240px 固定）
- 顶部无导航栏（侧边栏足够）
- 内容区最大宽度 1400px，居中
- 侧边栏菜单项：
  - 📊 仪表盘
  - 🔌 Providers
  - 🧩 模型管理
  - 🔀 路由管理
  - ⚙️ 设置
  - 📄 配置
  - 🔄 变更管理
  - 💬 会话
  - 📈 统计分析
  - 📋 日志

## 通用组件
- Toast 通知：右上角弹出，3 秒自动消失，支持 success/error/warning/info
- 确认弹窗：居中 Modal，标题 + 描述 + 确认/取消按钮
- 加载态：骨架屏（脉冲动画灰色块）
- 空状态：居中图标 + 标题 + 描述 + 引导操作按钮
- 错误态：红色边框卡片 + 错误信息 + 重试按钮
- 分页器：底部居中，页码 + 上一页/下一页
- API Key 显示：默认 `****` + 最后 4 位 + 👁 切换按钮
```

---

> 提示词版本：v1.0 | 适用项目：Moon Bridge | 后端 API 基路径：`/api/v1`
