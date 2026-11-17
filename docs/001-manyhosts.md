# 001 ManyHosts

ManyHosts 是基于 [mygo](https://github.com/idealisan/mygo) 原生 UI 的 hosts 配置管理工具。
配置直接存放在 hosts 文件中，不再使用额外的 JSON 配置文件。

## 分块格式

每个配置在 hosts 文件中是一个块：

```
#==hosts-profile-<profile-name>-start

<content>
#==hosts-profile-<profile-name>-end
```

启动时，hosts 中不属于任何块的非空内容会自动生成名为「原始内容 / 原始内容 2 …」
的块并显示在列表中（默认启用），不会丢弃；块间空白行保持为普通文本以维持排版。
关闭配置时，内容每行以 `# ` 开头；开启时取消注释。
旧格式（`;` 标记和 `; ` 注释）仍可读取，但写回时会统一改成 `#`。
块标记中的 `hosts-profile` 是历史数据格式，未随项目更名为 `manyhosts`，以免已有配置失效。

## 界面与交互

- 主窗口上方是「新增 / 删除 / 编辑」三个按钮，下方是一个表格。
- 表格列：`启用`（低饱和度绿色开关，可多选）、`名称`（可就地重命名）、`行数`。
- 开关即时生效：拨动后立即重写 hosts 文件，开启的块取消注释，关闭的块加上注释。
- 编辑窗口是非模态的，只显示并编辑当前配置块的内容；保存后写回该块。
- 全部配置关闭时，hosts 里的块内容都被注释，不会删除块本身。

## 平台支持

- Windows：hosts 路径 `C:\Windows\System32\drivers\etc\hosts`，写文件用 CRLF；
  构建时嵌入 `requireAdministrator` manifest 并使用 GUI 子系统（无控制台）。
- macOS：hosts 路径 `/etc/hosts`，写文件用 LF。直接写入被拒绝时，会自动调用
  `osascript ... with administrator privileges` 弹出系统授权对话框完成写入。
- Linux：hosts 路径 `/etc/hosts`，写文件用 LF；需要 `sudo` 权限（没有 AppleScript 机制）。

## 运行与构建

开发运行（各平台通用，带控制台便于看日志）：

```sh
go run .
```

各平台构建：

```powershell
# Windows（自动按当前 CPU 架构选择 amd64 / arm64 的 manifest）
.\build.ps1
```

```sh
# macOS / Linux
./build.sh
```

开发或测试时可设置环境变量 `HOSTS_PROFILE_HOSTS` 指向项目内的 hosts 副本，
避免改动系统文件：

```sh
HOSTS_PROFILE_HOSTS=./tmp/test-hosts go run .
```

跨平台上任意一方都可以编译出其他平台的二进制（纯 Go，无 cgo）：

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o tmp/ManyHosts-windows-amd64.exe .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o tmp/manyhosts-darwin-arm64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o tmp/manyhosts-linux-amd64 .
```

## GitHub Actions

推送 `v*` 标签（例如 `git tag v0.1.0 && git push origin v0.1.0`）会触发
`.github/workflows/release.yml`：

- 交叉编译 Windows amd64/arm64、macOS amd64/arm64、Linux amd64/arm64 六个目标
- 上传为 GitHub Actions artifacts
- 自动创建 Release 并把六个二进制作为附件发布
- 手动触发 `workflow_dispatch` 只构建并上传 artifacts，不创建 Release

`build.ps1` 使用 `-ldflags "-H=windowsgui"`，并在构建期间临时把
`resources\manifest\admin_<arch>.syso`（由 `admin.manifest` 生成的
`requireAdministrator` manifest）放入包目录，构建完移回，避免 `go test`
也要求提权。`build.sh` 不附加 manifest。

## 注意

- 权限不足时状态栏会给出提示：Windows 请以管理员身份运行，macOS/Linux 请用 `sudo`。
- 当前版本尚未提供系统托盘、备份/恢复、配置导入导出，留待后续迭代。
