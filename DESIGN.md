# Folcon 设计文档

- 状态：设计稿（未开始实现）
- 日期：2026-11-05
- 目标平台：Windows 10 / 11 x64
- 技术栈：Go + MyGo（github.com/egoist/mygo）原生 UI

---

## 1. 概述

Folcon 是一个 Windows 小工具：把一个文件夹在资源管理器中的图标，改成该文件夹里某个应用程序的图标。

典型用法：绿色软件 / 便携应用解压到自己的目录，目录名往往不直观。选中这个目录，再选中里面的主程序 `app.exe`，目录图标就变成 `app.exe` 的图标，一眼能认出是什么软件。

支持一次选择多个文件夹，按向导逐个确认。每个文件夹确认后立即写入，然后自动进入下一个。

### 1.1 目标

- 单个或批量设置文件夹图标。
- 从文件夹内的 `.exe` 提取图标；一个 exe 有多个图标组时可指定。
- 也可以直接使用文件夹内现成的 `.ico` 文件。
- 纯原生界面、启动快、体积小、免安装。
- 不依赖 WebView2，不需要管理员权限（受保护目录除外）。

### 1.2 非目标（v1 明确不做）

- 撤销 / 备份 / 恢复原图标
- 历史记录、持久化、重新应用
- 安装包、自动更新、代码签名
- PNG / DLL / LNK 图标来源
- 自动扫描候选程序
- 批量结果汇总页
- 管理员提权
- 文件夹图标之外的任何系统定制

---

## 2. 已确认的产品决策

| 主题 | 决策 |
|---|---|
| 交互模型 | 选好文件夹后立即弹出文件选择器，逐个文件夹处理 |
| 应用发现 | 完全手动浏览，不做自动扫描 |
| 文件类型 | 只接受 `.exe` 和 `.ico` |
| 选择范围 | 只能选目标文件夹内部（含子目录）的文件 |
| 多图标 | 不做选择；自动使用最大分辨率的可用图标组 |
| 预览 | 不做预览页；选择文件后直接应用 |
| 应用时机 | 选择后立即写入 |
| 成功后 | 自动进入下一个文件夹，并再次弹出文件选择器 |
| 结束时 | 回到欢迎页，不显示汇总 |
| 覆盖策略 | 静默覆盖已有图标；保留 desktop.ini 其他键 |
| 撤销 | 不做 |
| 持久化 | 不做 |
| 目标限制 | 普通文件夹不限制；只拒绝技术上已知不可靠的位置 |
| 预检失败 | 提示原因，可跳过或取消整批 |
| 写入失败 | 立即中止整批，已应用的保留 |
| 空文件夹 | 提示，可跳过或取消 |
| 中途取消 | 取消文件选择器 = 跳过当前文件夹；结束整批时若已应用则确认不撤销 |
| 刷新 | `SHChangeNotify` + 自动 `ie4uinit.exe -show` |
| 界面语言 | 简体中文 |
| 主题 | 跟随系统深浅色 |
| 发布 | 便携单文件 exe，仅 x64 |
| 应用标识 | Folcon 0.1.0，`io.github.byte-me-labs.folcon`，文件夹 + 应用窗口图标 |

---

## 3. 用户流程

### 3.1 欢迎页

- 标题、简短说明。
- 主按钮：**选择文件夹…**（系统文件夹选择对话框，支持多选）。
- 拖放区：把一批文件夹拖进窗口，效果与选择文件夹相同；拖入的文件忽略。
- 没有文件夹被选中时，不进入向导。
- 重复路径去重，保持首次出现顺序。

### 3.2 选择与写入（每个文件夹）

选好文件夹后直接打开系统文件选择器，不再有需要手动点按钮进入的向导页：

1. 文件选择器默认定位到当前文件夹，过滤器 `.exe`、`.ico`；选择后校验文件必须位于当前文件夹内部（含子目录）。
2. 读取图标组，自动选择**最大尺寸最大**的可用组（并列时取序号小的，`.ico` 只有一个组），不显示预览或确认页，立即写入。`IconResource` 引用的是整个图标组，Explorer 会按视图大小自行取用组内相应尺寸；选择最大组即保证大图标视图能拿到最高分辨率。
3. 写入成功后自动处理下一个文件夹，再次弹出文件选择器。
4. 取消文件选择器：跳过当前文件夹继续；若尚未应用任何图标，则直接回到欢迎页。
5. 空文件夹 / 预检失败：显示原因，提供 **跳过此文件夹** / **结束向导**。
6. 写入失败：显示错误，中止整批，回到欢迎页。
7. 选择器打开期间，欢迎页显示当前进度（第 N 个、已应用数量）。

### 3.3 结束

- 最后一个文件夹处理成功 → 直接回到欢迎页。
- 不显示本轮汇总。
- 窗口关闭时若已应用过图标 → 先确认。

---

## 4. Windows 写入规格

### 4.1 desktop.ini 内容

目标文件 `desktop.ini`，编码 **UTF-16LE + BOM**，CRLF 换行。

最小内容：

```ini
[.ShellClassInfo]
IconResource=.\app.exe,1
ConfirmFileOp=0
```

- `IconResource` 使用**相对路径**，相对于 `desktop.ini` 所在文件夹。
- 索引：
  - `.exe`：所选图标组在资源枚举顺序中的序号（从 0 开始）。
  - `.ico`：恒为 `0`。
- `ConfirmFileOp=0`：仅当文件中不存在该键时写入，用于抑制“正在删除系统文件夹”的确认框。
- 若已有 `IconFile` / `IconIndex` 旧键，写入 `IconResource` 时删除它们。
- 保留文件中其他所有内容：[ViewState]、InfoTip、LocalizedResourceName、CLSID、其他节等。
- 若原文件无法解析，按行做保守合并：找到 `[.ShellClassInfo]` 节，替换/插入 `IconResource` 行，其余行原样保留。

### 4.2 文件与文件夹属性

1. 读取已有 `desktop.ini`（若有）。
2. 合并内容。
3. 若已有 `desktop.ini` 是只读，先清除只读。
4. 写入 `desktop.ini`（UTF-16LE + BOM）。
5. 设置 `desktop.ini` 属性为 `Hidden | System`。
6. 设置目标文件夹属性为 `ReadOnly`（保留已有的 Hidden/System 等其他位）。
7. 调用刷新。

**顺序不可颠倒**：实测先给文件夹加只读，再写 `desktop.ini` 会被拒绝（Access Denied）。

### 4.3 路径规则

- 相对路径必须是文件相对目标文件夹的路径，例如：
  - `.\app.exe`
  - `.\bin\app.exe`
  - `.\assets\icon.ico`
- 禁止出现 `..`。
- 选中的文件在目标文件夹之外 → 拒绝，停留在选择阶段。
- 文件路径若包含逗号或换行，无法安全写入 `IconResource`，拒绝并提示重新选择。
- 大小写不敏感比较。
- 若文件本身是符号链接 / 重解析点，解析真实路径后仍必须位于目标文件夹内。

### 4.4 图标组枚举与序号

- 用 `LoadLibraryExW(path, 0, LOAD_LIBRARY_AS_DATAFILE | LOAD_LIBRARY_AS_IMAGE_RESOURCE)` 以数据方式加载，不执行目标代码。
- 用 `EnumResourceNamesW(handle, RT_GROUP_ICON /* 14 */, ...)` 枚举图标组，记录：
  - 组标识（数字 ID 或字符串名）
  - 枚举顺序（即 `IconResource` 的序号）
  - 每个组包含的尺寸列表
- 序号语义已实测验证：
  - `powershell.exe` 有两组：序号 0 = 普通 PowerShell 图标，序号 1 = 带盾牌图标。
  - 写入 `IconResource=.\powershell.exe,1` 后，文件夹实际渲染为盾牌图标。
- 枚举顺序与 `ExtractIconEx` 的正数序号一致。

### 4.5 图标帧读取

- 用与枚举相同的模块句柄直接读取资源：`FindResourceW` → `SizeofResource` → `LoadResource` → `LockResource`。不经过 winres，避免其 MUI / 语言回退逻辑漏掉图标组（实测 notepad.exe 上 winres 只看到 5 个非图标资源）。
- 帧可能是完整 PNG（通常是 256×256）或 BMP/DIB；Folcon 只读取组内每帧的声明尺寸（宽高字节为 0 表示 256），用于选出分辨率最高的组。
- 引用的是整个图标组，不单独导出或渲染某一帧；Explorer 按视图大小从组内取用对应帧。
- `.ico` 文件解析 `ICONDIR` / `ICONDIRENTRY`，同样读取实际帧尺寸。

### 4.6 刷新

每次写入后：

1. `SHChangeNotify(SHCNE_UPDATEDIR /* 0x1000 */, SHCNF_PATHW /* 0x0005 */, folder, NULL)`
2. 隐藏窗口运行 `%SystemRoot%\System32\ie4uinit.exe -show`
3. 忽略 `ie4uinit` 的失败与退出码
4. 不重启 Explorer、不删除图标缓存数据库

界面同时显示提示：若 Explorer 窗口仍未更新，按 F5。

---

## 5. 预检与拒绝规则

进入每个文件夹时立即预检，失败则提示原因，提供 **跳过此文件夹** / **取消整个向导**。

### 5.1 拒绝的位置

| 情况 | 检测方式 | 原因 |
|---|---|---|
| OneDrive 目录 | 路径前缀匹配 `%OneDrive%` / `%OneDriveConsumer%` / `%OneDriveCommercial%` | `desktop.ini` 不会同步 |
| 网络共享 | UNC 前缀 `\\`；或驱动器类型为 `DRIVE_REMOTE` | Shell 图标/信任策略不可靠 |
| 超长路径 | 目标文件夹路径长度 + `\desktop.ini` 接近或超过 260 | Shell 通知与部分 API 受限 |
| 重解析点 | 目标文件夹带 `FILE_ATTRIBUTE_REPARSE_POINT` | 写入会作用到链接目标，行为不符合预期 |
| 无写权限 | 临时文件写入测试失败 | 无法创建 desktop.ini |
| 无属性写权限 | 对文件夹调用 `SetFileAttributesW`（写入当前属性）失败 | 无法设置 ReadOnly |
| desktop.ini 不可写 | 已存在但无法以写方式打开 | 无法合并写入 |

### 5.2 预检实现

- 临时文件测试：在目标文件夹创建 `.folcon-probe-<pid>-<rand>`，然后删除。
- 属性测试：用当前属性调用 `SetFileAttributesW`，不改变实际状态，只验证权限。
- 以上测试失败不留下残留文件。
- 若目标文件夹在向导过程中被删除或变成不可用，按写入失败处理（中止整批）。

### 5.3 不拒绝的情况

- 普通本地 NTFS / FAT / exFAT 文件夹。
- 已有 `desktop.ini` 的文件夹（静默合并覆盖）。
- 已有自定义图标的文件夹（静默覆盖）。
- 用户主目录以外的本地目录（包括 D 盘、移动硬盘）。
- 系统特殊文件夹不额外拦截；若权限不足，由预检失败处理。
- 驱动器根目录不额外拦截；通常因权限或路径规则失败，按预检结果处理。

---

## 6. 错误处理

| 场景 | 行为 |
|---|---|
| 文件夹内没有 `.exe` / `.ico` | 提示，可跳过或取消 |
| 预检失败 | 提示原因，可跳过或取消 |
| exe 没有任何图标组 | 当前离线页提示，要求重新选择；不写入 |
| ico 无法解析 / 没有帧 | 当前离线页提示，要求重新选择；不写入 |
| 选中的文件在文件夹外 | 当前离线页提示，要求重新选择 |
| 实际写入失败（属性、磁盘、权限竞态） | 显示错误（路径 + 原因），立即中止整批；已应用的保留 |
| 用户中途取消 / 关窗 | 若已应用过文件夹，弹确认；确认后回欢迎页 |
| `ie4uinit` 失败 | 忽略，仅保留 F5 提示 |
| 文件夹被外部删除 | 按写入失败处理 |

不做自动重试、不做管理员提权、不做回滚。

---

## 7. 技术架构

### 7.1 框架与依赖

- Go 1.27.3，使用本机工具链 `D:\DevTools\Golang\go1.27.3`（已验证 `go version go1.27.3 windows/amd64`，满足 mygo v0.8.3 的 Go 1.27+ 要求）。构建时显式使用该目录下的 `go.exe`，或把它的 `bin` 目录放在 PATH 前面，避免落到其他 Go 版本。
- `github.com/egoist/mygo` v0.8.3，原生 UI 包 `github.com/egoist/mygo/ui`。
- `golang.org/x/sys/windows` v0.48.0：`LoadLibraryEx` / `FindResource` / `LoadResourceData` / `FreeLibrary` 等类型安全封装。
- 无 cgo；`CGO_ENABLED=0` 必须可用。
- `EnumResourceNamesW` 没有 x/sys 封装，用一个 `NewLazySystemDLL("kernel32.dll")` 的 proc 加 `syscall.NewCallback` 调用；所有 DLL 都从 System32 解析，避免 DLL 劫持。
- 不依赖 WebView2。

### 7.2 模块划分

```text
main.go
internal/app/         状态机、欢迎页、批次与对话框流程
internal/winicon/     RT_GROUP_ICON 枚举、图标帧尺寸读取
internal/foldericon/  desktop.ini 读取/合并/写入、属性设置
internal/winpath/     路径规范化、内部路径校验、预检、不可用位置检测
internal/refresh/     SHChangeNotify、ie4uinit 调用
tools/genicon/        占位图标生成器
```

### 7.3 状态机

```text
Welcome
  └─ 选择/拖入文件夹 → 对每个 folder[i]：
       Preflight(folder[i])
       ├─ 不可用 / 空文件夹 → 跳过 / 结束
       └─ 可用 → 打开文件选择器
            ├─ 取消 → 跳过（未应用过则直接 Welcome）
            ├─ 选择文件 → 读取图标组
            │    ├─ 0 个可用组 → 提示后重新选择
            │    └─ 有可用组 → 自动选最大分辨率的组
            └─ 应用 → 成功 → 刷新 → i+1 或 Welcome
                     → 失败 → 错误 → Welcome（中止）
```

- 状态只存在内存中，退出即丢失。
- 已应用的文件夹不回滚。
- 单实例运行；第二次启动聚焦已有窗口。

### 7.4 MyGo 使用要点

- `main` 中 `mygo.App.WhenReady` 创建窗口；`App.Run()` 在主 goroutine。
- 文件对话框、文件 I/O、图标提取、`ie4uinit` 调用都放到后台 goroutine，不阻塞主线程。
- 结果通过 `Window.Update` 回到主线程更新界面。
- 状态放在应用自己的 struct 中；`ui.Element` 不跨构建持有；需要跨构建的控件用 `ui.Handle` / `c.Key`。
- 原生 UI 文本由 DirectWrite 渲染，中文自动使用系统字体，无需配置。

### 7.5 窗口与外观

- 标题：`Folcon`
- 单一欢迎页，内容区固定为 360×220（不含标题栏与边框）；不可调整大小，标题栏只保留关闭按钮（无最小化、无最大化），不随流程改变大小。
- 主题：跟随系统深浅色。
- 不保存窗口位置（不持久化）。
- 无托盘、无全局快捷键、无自动更新。

---

## 8. 实测记录（2026-11-05，Windows 10 19045）

全部实验在 `%TEMP%\folcon-tech` 下进行，未修改用户文件。

### 8.1 desktop.ini 生效条件

| 变体 | 结果 |
|---|---|
| 无 desktop.ini | 默认文件夹图标 |
| 绝对路径 + 文件夹只读 + ini 隐藏/系统 | 生效 |
| 相对 `app.exe` + 文件夹只读 | 生效 |
| 相对 `.\app.exe` + 文件夹只读 | 生效 |
| 绝对路径 + **文件夹不设只读** | **不生效** |
| 绝对路径 + 文件夹只读 + ini 普通可见 | 生效（Hidden/System 非必需） |
| `.\icon.ico,0` + 文件夹只读 | 生效 |
| 旧式 `IconFile` + `IconIndex` | 生效 |

### 8.2 属性顺序

- 先设 `attrib +r`，再写 `desktop.ini` → **Access Denied**。
- 先写 `desktop.ini`，再设 `attrib +r` → 正常。

### 8.3 多图标组

- `powershell.exe`：`MSH_MAIN`、`MSH_SECURITY` 两组，枚举顺序与 `ExtractIconEx` 序号一致。
- `IconResource=.\powershell.exe,0` → 普通图标；`,1` → 盾牌图标。
- `explorer.exe` 28 组、`mspaint.exe` 40 组、`SHELL32.dll` 330 组、`imageres.dll` 360 组，说明必须正确处理组数较多的 exe。

### 8.4 刷新

- 修改后新的 Shell 查询能立即读到新图标。
- 已打开的 Explorer 窗口可能仍需 F5；因此保留提示，并自动运行 `ie4uinit.exe -show`。

### 8.5 已知限制（来自资料 + 实测）

- OneDrive 不同步 `desktop.ini`。
- 网络路径受 2026 年 6 月 Windows 的 `desktop.ini` 信任策略影响。
- Shell 通知使用 `SHCNF_PATHW`，超长路径不可靠。
- 文件夹 ReadOnly 会让 Windows 把它当“系统文件夹”；用 `ConfirmFileOp=0` 规避删除确认。
- 隐藏的 `desktop.ini` + 文件夹属性修改是杀毒软件常见的敏感模式，可能被误报。
- exe 被重命名或删除后，图标会失效；整个文件夹移动则相对路径仍有效。

---

## 9. 测试计划

### 9.1 单元测试

- **INI 合并**
  - 没有 desktop.ini
  - 已有 `[ViewState]` / `InfoTip` / `LocalizedResourceName` / `CLSID`
  - 已有 `IconResource`、`IconFile` + `IconIndex`
  - UTF-16LE BOM、ANSI、损坏内容
  - 只读 desktop.ini
  - `ConfirmFileOp` 已存在 / 不存在
- **路径校验**
  - 内部文件、子目录文件、`..`、外部文件、符号链接指向外部
  - 大小写、Unicode、逗号、换行
  - OneDrive 前缀、UNC、`DRIVE_REMOTE`、长路径、重解析点
- **图标组**
  - 序号与 `ExtractIconEx` 一致
  - 字符串命名组、数字命名组
  - 缺失尺寸识别
- **属性**
  - 先 ini 后 folder 的顺序
  - 已有 Hidden/System 位的保留
  - 预检不留下临时文件

### 9.2 UI 测试（`ui.NewTester`）

- 欢迎页 → 添加两个文件夹 → 处理第一个 → 自动进入第二个 → 回欢迎页
- 空文件夹 → 跳过 → 继续
- 预检失败 → 跳过 / 取消
- 中途取消 → 确认弹窗
- 选择 exe → 自动使用最大分辨率的组 → 直接应用
- 写入失败 → 中止 → 回欢迎页

### 9.3 手工端到端

- 临时目录 + `powershell.exe` 的 0/1 两组
- `.ico` 文件
- 已有自定义图标的文件夹（验证静默覆盖且保留其他键）
- 无权限目录（如 Program Files 子目录）
- OneDrive / UNC / 超长路径（验证拒绝与跳过）
- 删除已设置图标的文件夹（验证无系统文件夹提示）
- Explorer 已打开时应用图标（验证 F5 提示与 `ie4uinit`）

---

## 10. 构建与发布

- 工具链：`D:\DevTools\Golang\go1.27.3\bin\go.exe`（Go 1.27.3）；构建前确认 `go version` 输出 `go1.27.3 windows/amd64`
- 应用名：`Folcon`
- exe 名：`Folcon.exe`
- 版本：`0.1.0`
- 标识符：`io.github.byte-me-labs.folcon`（见 `mygo.json`）
- 图标：程序生成的 1024×1024 PNG（文件夹 + 应用窗口徽标），放在 `resources/icon.png`
- 构建：`go tool mygo build`，取 `build/windows-amd64/Folcon.exe` 作为便携版发布；同目录的 `Folcon Setup 0.1.0.exe` 是 mygo 顺带生成的安装包，不随便携版发布
- 仅 x64；不做签名
- SmartScreen 可能提示未知发布者
- 发布物只有一个 exe，不含外部资源

---

## 11. 实施里程碑

1. **项目骨架**
   - `mygo init -template native`
   - 应用标识、图标、窗口、欢迎页
2. **图标读取**
   - RT_GROUP_ICON 枚举
   - 帧尺寸读取与最大组选择
   - 图标组序号测试
3. **写入核心**
   - desktop.ini 合并/写入
   - 属性设置顺序
   - SHChangeNotify + ie4uinit
   - 临时目录端到端验证
4. **向导与选择器**
   - 状态机
   - 文件夹队列、进度、自动下一个
   - 文件选择后自动应用最大组
5. **预检与错误**
   - 不可用位置检测
   - 跳过/取消/中止流程
   - 取消确认
6. **测试与打磨**
   - 单元测试、UI 测试
   - 真实 Explorer 验证
   - 便携构建

---

## 12. 待实现时确认的假设

- MyGo v0.8.3 的拖放 / 对话框组合能覆盖交互；欢迎页已通过 `ui.NewTester` 无头测试，仍需实机人工确认拖放、文件选择器与确认对话框。
- 图标组枚举顺序在所有 exe 上都与 `ExtractIconEx` 序号一致（已在 `powershell.exe` 验证，需再测 `mspaint.exe` / `explorer.exe`）。
- 系统自带 exe（如 `notepad.exe`）的图标可能位于 `zh-CN\notepad.exe.mui`，复制到普通文件夹后 `LoadLibraryEx` 找不到图标组。这类文件会在读取阶段被拒绝；常见的第三方应用把图标嵌在 exe 内，不受影响。
- `ie4uinit.exe -show` 在 Win10/11 上足够且不弹窗。
- 相对路径中包含 `;` / `#` 时 Windows 的 INI 解析行为（实现时补测；含逗号或换行的路径直接拒绝）。
- 使用 `D:\DevTools\Golang\go1.27.3` 的 Go 1.27.3 完成构建；构建前确认 `GOROOT` 指向该目录、`GOVERSION` 为 `go1.27.3`（均已验证）。
- 生成的图标可直接作为 mygo 的 config 图标（PNG，1024×1024）。
