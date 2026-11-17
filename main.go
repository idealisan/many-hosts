// ManyHosts：配置直接以分块形式存放在 hosts 文件里。
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	blockPrefix    = "#==hosts-profile-"
	oldBlockPrefix = ";==hosts-profile-" // 兼容旧格式
	blockStart     = "-start"
	blockEnd       = "-end"
	commentPrefix  = "# "
	oldComment     = "; " // 兼容旧格式
)

// Profile 是 hosts 文件中的一个配置块。
type Profile struct {
	Name    string
	Content string
	Enabled bool
}

// hostsPart 是 hosts 文件中的一段：普通文本或一个配置块。
type hostsPart struct {
	text    string
	profile *Profile
}

// app 持有主窗口与编辑窗口共享的状态。
type app struct {
	hostsPath string
	parts     []hostsPart

	table   ui.ListState
	current int

	editWin     *mygo.Window
	editProfile *Profile
	editName    string
	editDraft   string

	status string
}

// hostsFile 返回各平台的 hosts 文件路径。
func hostsFile() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}
	return "/etc/hosts"
}

// parseStart 解析配置块的起始标记，返回名称和所用的标记前缀。
func parseStart(line string) (string, string, bool) {
	for _, prefix := range []string{blockPrefix, oldBlockPrefix} {
		if strings.HasPrefix(line, prefix) && strings.HasSuffix(line, blockStart) {
			name := strings.TrimSuffix(strings.TrimPrefix(line, prefix), blockStart)
			if name != "" {
				return name, prefix, true
			}
		}
	}
	return "", "", false
}

// decodeBlock 从块内容中还原文本和启用状态。
// 关闭时每行以 "; " 开头，开启时保持原样。
func decodeBlock(lines []string) (bool, string) {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	isComment := func(line string) bool {
		return strings.HasPrefix(line, commentPrefix) || strings.HasPrefix(line, oldComment)
	}
	commented, found := true, false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		found = true
		if !isComment(line) {
			commented = false
			break
		}
	}
	if !found {
		commented = false
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if commented && isComment(line) {
			if strings.HasPrefix(line, commentPrefix) {
				out = append(out, strings.TrimPrefix(line, commentPrefix))
			} else {
				out = append(out, strings.TrimPrefix(line, oldComment))
			}
		} else {
			out = append(out, line)
		}
	}
	return !commented, strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// loadParts 读取 hosts 文件并拆成普通文本和配置块。
func loadParts(data string) []hostsPart {
	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	var parts []hostsPart
	var raw []string
	flush := func() {
		if len(raw) == 0 {
			return
		}
		parts = append(parts, hostsPart{text: strings.Join(raw, "\n")})
		raw = nil
	}
	for i := 0; i < len(lines); i++ {
		name, prefix, ok := parseStart(lines[i])
		if !ok {
			raw = append(raw, lines[i])
			continue
		}
		flush()
		end := prefix + name + blockEnd
		var content []string
		i++
		for ; i < len(lines) && lines[i] != end; i++ {
			content = append(content, lines[i])
		}
		if i >= len(lines) { // 缺少结束标记，按普通文本保留
			raw = append(raw, lines[i-len(content)-1])
			raw = append(raw, content...)
			continue
		}
		enabled, text := decodeBlock(content)
		parts = append(parts, hostsPart{profile: &Profile{Name: name, Content: text, Enabled: enabled}})
	}
	flush()
	return adoptRawParts(parts)
}

// adoptRawParts 把不属于任何块的非空内容包装成新块，纳入列表管理。
// 空白行保留为普通文本，以维持原有排版。
func adoptRawParts(parts []hostsPart) []hostsPart {
	used := make(map[string]bool)
	for _, part := range parts {
		if part.profile != nil {
			used[part.profile.Name] = true
		}
	}
	next := 1
	newName := func() string {
		for {
			name := "原始内容"
			if next > 1 {
				name = fmt.Sprintf("原始内容 %d", next)
			}
			next++
			if !used[name] {
				used[name] = true
				return name
			}
		}
	}

	out := make([]hostsPart, 0, len(parts)+2)
	for _, part := range parts {
		if part.profile != nil || strings.TrimSpace(part.text) == "" {
			out = append(out, part)
			continue
		}
		lines := strings.Split(part.text, "\n")
		first, last := 0, len(lines)-1
		for first < len(lines) && strings.TrimSpace(lines[first]) == "" {
			first++
		}
		for last >= first && strings.TrimSpace(lines[last]) == "" {
			last--
		}
		if first > last {
			out = append(out, part)
			continue
		}
		if first > 0 {
			out = append(out, hostsPart{text: strings.Join(lines[:first], "\n")})
		}
		content := strings.Join(lines[first:last+1], "\n")
		out = append(out, hostsPart{profile: &Profile{Name: newName(), Content: content, Enabled: true}})
		if last+1 < len(lines) {
			out = append(out, hostsPart{text: strings.Join(lines[last+1:], "\n")})
		}
	}
	return out
}

// encodeBlock 按分块格式写出一个配置。
func encodeBlock(p *Profile) []string {
	lines := []string{blockPrefix + p.Name + blockStart, ""}
	if p.Content != "" {
		for _, line := range strings.Split(strings.TrimRight(p.Content, "\n"), "\n") {
			if p.Enabled {
				lines = append(lines, line)
			} else if strings.TrimSpace(line) == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, commentPrefix+line)
			}
		}
	}
	return append(lines, blockPrefix+p.Name+blockEnd)
}

// writeHosts 把当前段落写回 hosts 文件。
func (a *app) writeHosts() error {
	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}
	var b strings.Builder
	for i, part := range a.parts {
		if i > 0 {
			b.WriteString(eol)
		}
		if part.profile == nil {
			b.WriteString(strings.TrimRight(part.text, "\n"))
		} else {
			b.WriteString(strings.Join(encodeBlock(part.profile), eol))
		}
	}
	data := b.String()
	if !strings.HasSuffix(data, eol) {
		data += eol
	}
	err := os.WriteFile(a.hostsPath, []byte(data), 0o644)
	if err != nil && runtime.GOOS == "darwin" && os.IsPermission(err) {
		return writeHostsWithAppleScript(a.hostsPath, data)
	}
	return err
}

// appleQuote 转义 AppleScript 字符串字面量。
func appleQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// writeHostsWithAppleScript 在 macOS 上通过系统授权对话框写入 hosts。
// 读取 /etc/hosts 不需要权限，因此只在写入被拒绝时使用。
func writeHostsWithAppleScript(path, data string) error {
	dir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "manyhosts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	staging := filepath.Join(dir, "hosts")
	if err := os.WriteFile(staging, []byte(data), 0o600); err != nil {
		return err
	}
	script := fmt.Sprintf(
		`do shell script "cp %s %s && chmod 644 %s" with administrator privileges`,
		appleQuote(staging), appleQuote(path), appleQuote(path),
	)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("系统授权写入失败：%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// save 写回 hosts 并显示结果。
func (a *app) save(message string) {
	if err := a.writeHosts(); err != nil {
		if os.IsPermission(err) {
			if runtime.GOOS == "windows" {
				a.setStatus("权限不足：请以管理员身份运行本程序")
			} else {
				a.setStatus("权限不足：请用 sudo 运行本程序")
			}
		} else {
			a.setStatus(fmt.Sprintf("写入 hosts 失败：%v", err))
		}
		return
	}
	a.setStatus(message)
}

func (a *app) setStatus(message string) {
	a.status = message
}

// profiles 返回文件中的所有配置，顺序与文件一致。
func (a *app) profiles() []*Profile {
	out := make([]*Profile, 0, len(a.parts))
	for _, part := range a.parts {
		if part.profile != nil {
			out = append(out, part.profile)
		}
	}
	return out
}

func (a *app) clampCurrent() {
	n := len(a.profiles())
	if n == 0 {
		a.current = -1
	} else if a.current >= n {
		a.current = n - 1
	}
}

func (a *app) currentProfile() *Profile {
	profiles := a.profiles()
	if a.current < 0 || a.current >= len(profiles) {
		return nil
	}
	return profiles[a.current]
}

func (a *app) hasName(name string) bool {
	for _, p := range a.profiles() {
		if p.Name == name {
			return true
		}
	}
	return false
}

// profileExists 判断编辑窗口指向的配置是否仍在 hosts 文件中。
func (a *app) profileExists(target *Profile) bool {
	if target == nil {
		return false
	}
	for _, p := range a.profiles() {
		if p == target {
			return true
		}
	}
	return false
}

// lineCount 统计配置内容的行数。
func lineCount(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// addProfile 在文件末尾新增一个默认关闭的空配置。
func (a *app) addProfile() {
	name := "配置 1"
	for i := 1; a.hasName(name); i++ {
		name = fmt.Sprintf("配置 %d", i+1)
	}
	a.parts = append(a.parts, hostsPart{profile: &Profile{Name: name, Content: "", Enabled: false}})
	a.current = len(a.profiles()) - 1
	a.save("已新增：" + name)
	a.openEdit()
}

// deleteProfile 删除当前配置块。
func (a *app) deleteProfile() {
	p := a.currentProfile()
	if p == nil {
		a.setStatus("请先选择一个配置")
		return
	}
	name := p.Name
	for i, part := range a.parts {
		if part.profile == p {
			a.parts = append(a.parts[:i], a.parts[i+1:]...)
			break
		}
	}
	a.clampCurrent()
	a.save("已删除：" + name)
}

// openEdit 打开非模态编辑窗口，只显示当前配置块的内容。
func (a *app) openEdit() {
	p := a.currentProfile()
	if p == nil {
		a.setStatus("请先选择一个配置")
		return
	}
	if a.editWin != nil {
		a.editWin.Focus()
		return
	}
	a.editProfile = p
	a.editName = p.Name
	a.editDraft = p.Content
	a.editWin = mygo.NewWindow(mygo.WindowOptions{
		Title:   "编辑配置",
		Width:   640,
		Height:  520,
		Content: ui.View(a.editView),
	})
	a.editWin.OnClosed(func() { a.editWin = nil })
}

func (a *app) closeEdit() {
	if a.editWin != nil {
		a.editWin.Close()
	}
}

// saveEdit 把名称和编辑内容写回对应配置块。
func (a *app) saveEdit() {
	if !a.profileExists(a.editProfile) {
		a.setStatus("该配置已被删除")
		a.closeEdit()
		return
	}
	name := strings.TrimSpace(a.editName)
	if name == "" {
		a.setStatus("配置名称不能为空")
		return
	}
	for _, p := range a.profiles() {
		if p != a.editProfile && p.Name == name {
			a.setStatus("配置名已存在：" + name)
			return
		}
	}
	a.editProfile.Name = name
	a.editProfile.Content = a.editDraft
	a.save("已保存：" + name)
	a.closeEdit()
}

// greenSwitch 是自绘的开关：用低饱和度绿色与表格行的选中高亮色区分。
func greenSwitch(c *ui.Context, key string, on *bool, label string) ui.Element {
	sw := ui.SwitchBase(c.Key(key), on).Size(32, 18).Radius(9).Label(label)
	var target float32
	if *on {
		target = 1
	}
	pos := sw.Animate("knob", target, 140*time.Millisecond)
	off := ui.RGB(165, 168, 172)
	active := ui.RGB(86, 145, 108) // 低饱和度绿色
	sw.Background(off.Mix(active, pos))
	sw.Draw(func(p *ui.Painter, r ui.Rect) {
		in := float32(2)
		d := r.H - 2*in
		knob := ui.Rect{X: r.X + in + pos*(r.W-r.H), Y: r.Y + in, W: d, H: d}
		p.Shadow(knob, d/2, 0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.25))
		p.Fill(knob, ui.RGB(255, 255, 255), d/2)
	})
	return sw
}

// mainView 是主窗口：上方按钮，下方表格。
func (a *app) mainView(c *ui.Context) {
	a.clampCurrent()
	// EditableText 在表格行内依赖 ListState.Selected 判断当前行。
	a.table.Selected = &a.current

	cols := []ui.TableColumn{
		{Title: "启用", Width: 64},
		{Title: "名称"},
		{Title: "行数", Width: 72, Align: ui.End},
	}

	ui.Column(c).Fill().Padding(12).Gap(8).OnClick(func() { a.current = -1 }).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Button(c, "新增").OnClick(a.addProfile)
			ui.Button(c, "删除").OnClick(a.deleteProfile)
			ui.Button(c, "编辑").OnClick(a.openEdit)
			ui.Text(c, "").Grow(1)
			ui.Text(c, a.status).FontSize(12).TextColor(c.Theme().TextMuted)
		})

		ui.Table(c, &a.table, cols, len(a.profiles()), func(row, col int) {
			p := a.profiles()[row]
			switch col {
			case 0:
				greenSwitch(c, "switch-"+strconv.Itoa(row), &p.Enabled, "启用 "+p.Name).
					OnChange(func() { a.save("已更新：" + p.Name) })
			case 1:
				ui.EditableText(c.Key("name-"+strconv.Itoa(row)), &p.Name).OnChange(func() { a.save("已重命名") })
			case 2:
				ui.Text(c, strconv.Itoa(lineCount(p.Content)))
			}
		}).Grow(1).OnSubmit(a.openEdit)
	})
}

// editView 是非模态编辑窗口：可编辑当前配置块的名称和内容。
func (a *app) editView(c *ui.Context) {
	ui.Column(c).Fill().Padding(16).Gap(12).Children(func() {
		if !a.profileExists(a.editProfile) {
			ui.Text(c, "该配置已被删除")
			ui.Button(c, "关闭").OnClick(a.closeEdit)
			return
		}
		ui.Text(c, "名称").FontSize(12).TextColor(c.Theme().TextMuted)
		ui.TextInput(c.Key("edit-name"), &a.editName).Placeholder("配置名称")
		ui.Text(c, "内容").FontSize(12).TextColor(c.Theme().TextMuted)
		ui.TextArea(c.Key("editor"), &a.editDraft).Grow(1)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			ui.Button(c, "取消").OnClick(a.closeEdit)
			ui.PrimaryButton(c, "保存").OnClick(a.saveEdit)
		})
	})
}

func main() {
	a := &app{
		hostsPath: hostsFile(),
		current:   -1,
		status:    "就绪",
	}
	// 便于开发与测试时指向项目内的 hosts 副本，避免改动系统文件。
	if path := os.Getenv("HOSTS_PROFILE_HOSTS"); path != "" {
		a.hostsPath = path
	}
	data, err := os.ReadFile(a.hostsPath)
	if err != nil {
		log.Printf("读取 hosts 失败：%v", err)
	} else {
		a.parts = loadParts(string(data))
	}
	a.clampCurrent()

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:    "ManyHosts",
			Width:    620,
			Height:   460,
			Content:  ui.View(a.mainView),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
