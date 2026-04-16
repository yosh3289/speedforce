# SpeedForce ⚡

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE) &nbsp; [English](README_eng.md)

轻量级 Windows 托盘应用，实时监测你的网络能否访问 Claude、OpenAI 和 Gemini —— 专为通过代理访问 AI 服务的用户打造。

🔵 全部正常 &nbsp; 🟡 部分降级 &nbsp; 🔴 严重异常

## 功能特性

- **6 路 HTTPS 连通探测**（Claude / OpenAI / Gemini 各 API + 网页端）
- **IP 信息展示** — 出口公网 IP + 局域网 IP + 国家/城市/ISP
- **官方服务状态** — Anthropic 和 OpenAI Statuspage 集成；Gemini 一键跳转 AI Studio 状态页
- **按服务粒度通知** — 自选关心的服务，挂了立即 Windows 弹窗提醒，5 分钟抖动冷却防刷屏
- **自适应检测频率** — 平时 60 秒一次，延迟突变或服务中断时自动切到 10 秒快速模式，连续 3 次稳定后恢复
- **智能代理支持** — 自动读取 Windows 系统代理，也可手动指定代理地址
- **中英双语** — 一键切换界面语言
- **日志导出** — 最近 7 天探测日志一键打包 zip，方便排查问题
- **单实例守护** — 防止重复启动；支持开机自启
- **轻量低耗** — 后台常驻 ~10 MB 内存，打开详情窗口 ~30 MB

## 快速开始

从 [Releases 页面](https://github.com/yosh3289/speedforce/releases) 下载最新的 `speedforce-vX.Y.Z-windows-amd64.zip`，解压后双击 `speedforce.exe` 即可运行。

启动后右下角托盘会出现闪电图标：
- **鼠标悬停** — 显示 IP、在线服务数量摘要
- **左键/右键** — 打开菜单（显示详情 / 设置 / 退出）
- **显示详情** — 查看每个服务的延迟、状态码、官方平台状态

## 配置

配置文件位于 `%APPDATA%\SpeedForce\config.yaml`，首次运行自动生成默认配置，修改后**实时热重载**，无需重启。

常用配置项均可通过**设置窗口**修改（代理、检测频率、通知开关、语言、开机自启）。高级配置直接编辑 YAML 文件。

完整配置说明见[设计文档](docs/superpowers/specs/2026-04-15-speedforce-design.md)。

## 从源码构建

```bash
git clone https://github.com/yosh3289/speedforce.git
cd speedforce
go build -ldflags="-H windowsgui -s -w" -o speedforce.exe ./cmd/speedforce
```

**依赖：**
- Go 1.22+
- C 编译器（fyne 的 OpenGL 驱动需要 CGO）— Windows 上安装 [MinGW-w64](https://github.com/brechtsanders/winlibs_mingw/releases/latest)

## 开发

```bash
# 运行测试
go test ./...

# 覆盖率报告
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

调试参数：
- `--tick=5` — 覆盖检测间隔（秒）
- `--fake-down="Claude API,OpenAI API"` — 模拟指定服务宕机

## 技术架构

```
┌──────────────────────────────────────┐
│          SpeedForce (Go)             │
│                                      │
│  Tray (systray) ←→ Core ←→ UI (fyne)│
│                     ↓                │
│     HTTPSProber / IPProber / Status  │
│                     ↓                │
│       Proxy-Aware HTTP Client        │
└──────────────────────────────────────┘
```

- **三层解耦** — UI / Core / 探测层通过接口分离，方便测试和未来换 UI 库
- **StateBus 订阅模型** — 所有探测结果汇入单一状态总线，UI 订阅更新
- **跨平台预留** — 平台相关代码（代理检测、开机自启）通过 Go build tag 隔离，macOS/Linux stub 已就位

## 路线图

- macOS + Linux 支持
- 持久化历史记录（SQLite）
- 延迟趋势迷你图
- 自定义探测端点
- 主题包

详见[设计文档 §12](docs/superpowers/specs/2026-04-15-speedforce-design.md#12-future-work-post-v01)。

## 致谢

- 托盘闪电图标来自 [Icons8 flat-color-icons](https://github.com/icons8/flat-color-icons)（MIT 许可）

## 许可证

MIT. 详见 [LICENSE](LICENSE)。
