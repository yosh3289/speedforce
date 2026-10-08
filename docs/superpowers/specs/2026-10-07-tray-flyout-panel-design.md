# 托盘弹出面板（取代详情窗口）设计

**日期**：2026-10-07 · **状态**：已与用户确认，实现第一版

参考 win-codexbar：左键托盘图标弹出一个贴着任务栏的深色无边框面板，点别处自动收起。面板**完全取代**原来的详情窗口。

## 交互

- 左键托盘图标：面板关着就打开，开着就收起（toggle）。
- 防抖：点托盘图标会先让面板失焦收起（鼠标按下），紧接着同一次点击的 tap（鼠标抬起）又会打开它。若面板在 300ms 内刚因失焦收起，忽略这次 tap。
- 位置：取点击时光标所在显示器，比较显示器区域与工作区判断任务栏在哪条边（下/上/左/右；自动隐藏的任务栏按“下”处理），面板贴着该边、与工作区留 8（逻辑像素）间距；沿任务栏方向以光标为中心，夹在工作区内。全部使用物理像素（GLFW 进程为 per-monitor DPI aware）。
- 收起：应用失去前台（fyne `Lifecycle.SetOnExitedForeground`）或按 Esc。只隐藏不销毁（避免 v0.1.5 那类“关闭后打不开”的问题和 GL 资源反复创建）。
- 右键菜单不变（显示详情 / 设置 / 退出），“显示详情”打开面板（show，不 toggle）。
- 面板不出现在任务栏 / Alt-Tab（`WS_EX_TOOLWINDOW`），Windows 11 上圆角（DWM，尽力而为）。

## 内容（深色，仅面板套深色主题）

1. 头部：`SpeedForce ⚡  6/6 在线` + 右侧检测模式；第二行 `公网 IP · 国家`。
2. 连接状态：6 行，名称列对齐 + 延迟条 + `xxx ms`。延迟条按 0–3000ms 线性显示；颜色沿用原规则：正常蓝、>3000ms 黄、失败红（失败行显示“失败”，条满格红色）。
3. 官方状态：每源一行，圆点颜色沿用原规则。
4. `▸ 网络详情`（可展开，类似 codexbar 的 Usage details）：局域网 IP、城市 / ISP、各路 HTTP 状态码。展开/收起后面板重新计算高度并重新定位。
5. 底部：`[Claude 状态] [OpenAI 状态] [Gemini 状态]` + 设置图标按钮。

宽 340（逻辑像素），高度随内容。状态更新时整体重建内容；仅当高度变化时重新定位。

## 结构

- `internal/ui/panel`：`panel.go`（窗口生命周期、Toggle/Show/FocusLost、防抖、订阅 StateBus；所有窗口操作在 fyne 主线程）、`view.go`（由 `core.State` 构建界面，纯构建、可测）、`theme.go`（深色主题，`container.NewThemeOverride`）。
- `internal/platform/flyout.go`：纯函数 `PlaceFlyout`（可在任意平台测试）；`flyout_windows.go`：`PositionFlyout(hwnd)`、`StyleFlyout(hwnd)`；其他平台空实现。HWND 通过 fyne `driver.NativeWindow.RunNative` 获取。
- `internal/ui/tray`：`Callbacks` 增加 `OnTap`（左键）；菜单“显示详情”仍走 `OnDetail`。
- 删除 `internal/ui/detail`；状态页按钮表与其测试迁入 `panel`。i18n 键由 `detail.*` 改为 `panel.*`。

## 测试

- `PlaceFlyout`：任务栏四个方向、贴边夹取、自动隐藏任务栏。
- 面板：tap 打开/再 tap 收起；失焦收起后 300ms 内的 tap 被忽略、之后的 tap 打开；Esc 收起。
- 视图：中英文文案、三个状态按钮网址、失败行、延迟条比例与颜色。
- 托盘：左键调用 `OnTap`，且不阻塞消息泵。
- 真实程序端到端：模拟左键 → 面板出现在任务栏旁；失焦 → 收起；再左键 → 打开；托盘探测正常。推送前跑与 CI 相同的 golangci-lint。
