# GLM 使用量监控

一个轻量级的智谱 AI (bigmodel.cn) GLM API 使用量桌面监控工具。以悬浮窗形式实时展示额度消耗情况，帮助开发者随时掌握 API 调用配额。

## 功能特性

- **悬浮窗监控** — 置顶透明悬浮条，始终可见，不干扰日常工作
- **悬停展开详情** — 鼠标悬停自动展开详细数据面板，离开自动收起
- **系统托盘** — 最小化到托盘图标，右键菜单支持刷新、设置、退出
- **定时刷新** — 可选 3/5/10 分钟自动轮询，也可手动立即刷新
- **多项指标显示** — 支持自定义主界面显示项（最多 4 项）

### 监控数据

| 指标 | 说明 |
|------|------|
| 本轮额度 | 5 小时周期内的 Token 使用百分比及剩余 |
| 每周额度 | 周维度 Token 使用百分比及剩余 |
| MCP 调用 | 月度 MCP 调用次数（已用 / 总量 / 剩余 / 使用率） |
| 套餐等级 | 当前账号套餐级别 |
| 下次刷新 | 本轮额度重置时间 |

### 可配置项

- **API Key** — 智谱开放平台 API Key（用于鉴权查询）
- **刷新频率** — 3 分钟 / 5 分钟 / 10 分钟
- **排版模式** — 单行紧凑 / 多行堆叠
- **窗口模式** — 悬浮窗 / 仅托盘图标
- **显示项** — 自定义主界面显示的指标（最多 4 项）
- **毛玻璃效果** — 开启/关闭透明模糊背景
- **开机自启** — 通过注册表设置跟随 Windows 启动

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.23, [Wails v2](https://wails.io/) |
| 前端 | Vue 3 + TypeScript + Tailwind CSS |
| 构建 | Vite |
| 系统集成 | fyne.io/systray（系统托盘）, Windows API（任务栏隐藏） |

## 项目结构

```
glm-monitor/
├── main.go              # Wails 应用入口，窗口配置
├── app.go               # 核心逻辑：API 请求、配置管理、托盘、轮询
├── go.mod / go.sum      # Go 依赖
├── wails.json           # Wails 项目配置
├── build/               # 构建资源（图标、安装器、清单）
└── frontend/
    ├── index.html
    ├── package.json
    ├── vite.config.ts
    └── src/
        ├── main.ts                       # Vue 入口
        ├── style.css                     # 全局样式
        ├── App.vue                       # 根组件
        └── components/
            ├── MonitorWidget.vue         # 悬浮监控条 + 详情面板
            └── SettingsPanel.vue         # 设置界面
```

## 配置文件

程序配置保存在 `%APPDATA%\GLM-Monitor\config.json`，包含：

```json
{
  "api_key": "",
  "refresh_interval": 5,
  "layout_mode": "single-line",
  "display_items": ["5h_used", "next_refresh"],
  "window_width": 200,
  "glass_mode": true,
  "auto_start": false,
  "window_mode": "floating"
}
```

## 开发

### 前置要求

- Go 1.23+
- Node.js 16+
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

### 开发模式

```bash
wails dev
```

启动后自动运行 Vite 开发服务器，前端修改支持热重载。

### 构建发布

```bash
wails build
```

生成的可执行文件位于 `build/bin/` 目录。

## API 说明

程序调用智谱 AI 的配额查询接口：

```
GET https://open.bigmodel.cn/api/monitor/usage/quota/limit
Authorization: <API Key>
```

返回数据包含当前账号的 Token 限额（5 小时周期 / 周维度）和 MCP 调用限额。

## 使用方法

1. 启动程序后，悬浮窗默认显示在屏幕上
2. 右键悬浮窗 → **设置**，填入智谱 API Key 并保存
3. 数据将按设定的刷新频率自动更新
4. 鼠标悬停悬浮窗可查看完整详情
5. 拖拽悬浮窗边缘可调整宽度，自动保存
