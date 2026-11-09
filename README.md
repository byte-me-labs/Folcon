# Folcon

把文件夹里某个应用的图标，设为这个文件夹在 Windows 资源管理器中的图标。

Folcon 使用 [MyGo](https://github.com/egoist/mygo) 原生 UI，无 WebView、无后台常驻、不写注册表，默认生成单文件 x64 程序。

## 功能

- 批量选择文件夹，选完后逐个为每个文件夹选择图标来源
- 支持 `.exe` 和 `.ico`
- 自动使用图标文件中分辨率最高的图标组，不需要手动挑选
- 图标源必须位于目标文件夹内，`IconResource` 使用相对路径
- 保留 `desktop.ini` 中的其他键，写入 `ConfirmFileOp=0`
- 写入后立即通知资源管理器刷新
- 支持把文件夹拖进窗口
- 简体中文界面

## 使用

1. 从 [Releases](https://github.com/byte-me-labs/Folcon/releases/latest) 下载 `Folcon.exe`。
2. 运行后点击“选择文件夹…”，或把一个/多个文件夹拖到窗口。
3. 在文件对话框中选择该文件夹内的 `.exe` 或 `.ico`。
4. 图标会立即应用；批量选择时自动进入下一个文件夹。
5. 某个文件夹没有可用文件或预检失败时，可以跳过或结束向导。

关闭窗口即退出；程序不保留配置，也不会在后台运行。

## 限制

- 仅支持 Windows 10/11 x64
- 图标源必须是目标文件夹内的 `.exe` 或 `.ico`，可以位于子文件夹
- 不支持磁盘根目录、网络位置、OneDrive 同步文件夹、符号链接/联接点，以及超过 260 个字符的路径
- 再次应用会直接覆盖原有的 `IconResource` / `IconFile` / `IconIndex`，没有撤销
- 文件夹会被设为只读，这是 Windows 让 `desktop.ini` 生效的方式

## 从源码构建

需要 Go 1.27.3，不需要 cgo：

```powershell
go tool mygo build -skip-installer
```

产物在 `build\windows-amd64\Folcon.exe`。去掉 `-skip-installer` 会额外生成 NSIS 安装包。

运行测试：

```powershell
go test ./...
```

重新生成应用图标：

```powershell
go run .\tools\genicon
```

## 发布

在 GitHub 上发布 release 后，`Release build` 工作流会在 `windows-latest` 上构建并上传 `Folcon.exe` 到该 release。

实现细节和设计取舍见 [DESIGN.md](DESIGN.md)。
