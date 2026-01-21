package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/moyu-x/classified-file/internal"
)

func (m *model) View() string {
	switch m.state {
	case StateConfig:
		return m.configView()
	case StateCounting:
		return m.countingView()
	case StateProcessing:
		return m.processingView()
	case StateComplete:
		return m.completeView()
	default:
		return "未知状态"
	}
}

func (m *model) configView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📦 文件分类去重工具 v1.0.0") + "\n\n")

	b.WriteString(separatorStyle.Render(strings.Repeat("─", 60)) + "\n\n")

	b.WriteString(labelStyle.Render("1. 选择重复文件处理方式：") + "\n")
	if m.focus == FocusMode {
		b.WriteString(focusedStyle.Render(m.modeList.View()) + "\n\n")
	} else {
		b.WriteString(normalStyle.Render(m.modeList.View()) + "\n\n")
	}

	if m.mode == internal.ModeMove {
		b.WriteString(labelStyle.Render("2. 输入移动目标目录：") + "\n")
		if m.focus == FocusTargetDir {
			b.WriteString(focusedStyle.Render(m.targetInput.View()) + "\n\n")
		} else {
			b.WriteString(normalStyle.Render(m.targetInput.View()) + "\n\n")
		}
	}

	label := m.getTargetDirLabel() + "输入要扫描的目录："
	b.WriteString(labelStyle.Render(label) + "\n")
	if m.focus == m.getFocusForDirInput() {
		b.WriteString(focusedStyle.Render(m.dirInput.View()) + "\n\n")
	} else {
		b.WriteString(normalStyle.Render(m.dirInput.View()) + "\n\n")
	}

	b.WriteString(labelStyle.Render("已添加目录列表：") + "\n")
	if m.focus == FocusDirList {
		b.WriteString(focusedStyle.Render(m.dirList.View()) + "\n\n")
	} else {
		b.WriteString(normalStyle.Render(m.dirList.View()) + "\n\n")
	}

	b.WriteString(separatorStyle.Render(strings.Repeat("─", 60)) + "\n")
	b.WriteString(hintStyle.Render("操作提示：") + "\n")
	b.WriteString("  • Tab 键切换焦点\n")
	b.WriteString("  • Enter 确认选择/添加目录\n")
	b.WriteString("  • Delete 删除已添加的目录\n")
	b.WriteString("  • Ctrl+C 退出程序\n")

	return lipgloss.NewStyle().
		Padding(1).
		Render(b.String())
}

func (m *model) countingView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔍 正在计算文件数量...") + "\n\n")
	b.WriteString(m.spinner.View() + "\n")
	b.WriteString("  正在遍历目录并统计文件数量...\n")
	b.WriteString("  已添加目录: " + strings.Join(m.scanDirs, ", "))

	return lipgloss.NewStyle().
		Padding(2).
		Render(b.String())
}

func (m *model) processingView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔄 正在处理文件...") + "\n\n")

	b.WriteString(labelStyle.Render("处理进度：") + "\n")
	b.WriteString(m.progressBar.View() + "\n\n")

	b.WriteString(statsBoxStyle.Render(
		m.renderStats(),
	) + "\n\n")

	b.WriteString(labelStyle.Render("当前文件：") + "\n")
	b.WriteString(filePathStyle.Render(m.currentFile) + "\n\n")

	return lipgloss.NewStyle().
		Padding(2).
		Render(b.String())
}

func (m *model) completeView() string {
	var b strings.Builder

	b.WriteString(successTitleStyle.Render("✅ 处理完成！") + "\n\n")

	b.WriteString(statsBoxStyle.Render(
		m.renderFinalStats(),
	) + "\n\n")

	b.WriteString(separatorStyle.Render(strings.Repeat("─", 60)) + "\n")
	b.WriteString(hintStyle.Render("按 Enter 继续扫描新目录，Ctrl+C 退出") + "\n")

	return lipgloss.NewStyle().
		Padding(2).
		Render(b.String())
}
