package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
	"github.com/moyu-x/classified-file/scanner"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.state == StateConfig {
			return m.updateConfigPhase(msg)
		}

	case tea.WindowSizeMsg:
		m.handleResize(msg)

	case countFilesMsg:
		m.totalFiles = msg.total
		m.state = StateProcessing
		return m, tea.Batch(
			m.startProcessing(),
			progressTick(),
		)

	case progressMsg:
		m.processed = msg.processed
		m.stats.Added = msg.added
		m.stats.Deleted = msg.deleted
		m.stats.Moved = msg.moved
		m.currentFile = msg.currentFile

		if m.totalFiles > 0 {
			percent := float64(m.processed) / float64(m.totalFiles)
			cmds = append(cmds, m.progressBar.SetPercent(percent))
		}

		m.logProgress()

		return m, tea.Batch(cmds...)

	case processCompleteMsg:
		m.state = StateComplete
		m.stats.EndTime = time.Now()
		m.logFinalStats()
		return m, nil

	case errMsg:
		m.err = msg
		return m, nil

	case spinnerTickMsg:
		if m.state == StateCounting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.state == StateConfig {
		var cmd tea.Cmd
		m.modeList, cmd = m.modeList.Update(msg)
		cmds = append(cmds, cmd)

		m.targetInput, cmd = m.targetInput.Update(msg)
		cmds = append(cmds, cmd)

		m.dirInput, cmd = m.dirInput.Update(msg)
		cmds = append(cmds, cmd)

		m.dirList, cmd = m.dirList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.state == StateProcessing {
		model, cmd := m.progressBar.Update(msg)
		m.progressBar = model.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateConfigPhase(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if msg.String() == "tab" {
		m.nextFocus()
		m.updateFocusState()
		return m, nil
	}

	if msg.String() == "enter" {
		return m.handleEnterKey()
	}

	if (msg.String() == "delete" || msg.String() == "backspace") && m.focus == FocusDirList && m.dirList.Index() >= 0 {
		m.dirList.RemoveItem(m.dirList.Index())
	}

	return m, tea.Batch(cmds...)
}

func (m *model) nextFocus() {
	switch m.focus {
	case FocusMode:
		m.focus = FocusTargetDir
	case FocusTargetDir:
		m.focus = FocusDirInput
	case FocusDirInput:
		m.focus = FocusDirList
	case FocusDirList:
		m.focus = FocusMode
	}
}

func (m *model) updateFocusState() {
	m.modeList.KeyMap.CursorUp.SetEnabled(m.focus == FocusMode)
	m.modeList.KeyMap.CursorDown.SetEnabled(m.focus == FocusMode)

	if m.focus == FocusTargetDir {
		m.targetInput.Focus()
	} else {
		m.targetInput.Blur()
	}

	if m.focus == FocusDirInput {
		m.dirInput.Focus()
	} else {
		m.dirInput.Blur()
	}

	m.dirList.KeyMap.CursorUp.SetEnabled(m.focus == FocusDirList)
	m.dirList.KeyMap.CursorDown.SetEnabled(m.focus == FocusDirList)
}

func (m *model) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusMode:
		if m.modeList.Index() == 0 {
			m.mode = internal.ModeDelete
		} else {
			m.mode = internal.ModeMove
		}
		return m, nil

	case FocusTargetDir:
		m.targetDir = m.targetInput.Value()
		return m, nil

	case FocusDirInput:
		dirPath := m.dirInput.Value()
		if dirPath != "" {
			m.scanDirs = append(m.scanDirs, dirPath)
			m.dirList.InsertItem(len(m.scanDirs)-1, dirItem{path: dirPath})
			m.dirInput.Reset()
		}
		return m, nil

	case FocusDirList:
		if len(m.scanDirs) > 0 {
			m.state = StateCounting
			return m, tea.Batch(
				spinnerTick(),
				countFilesCmd(m.scanDirs),
			)
		}
		return m, nil
	}

	return m, nil
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	width, _ := msg.Width, msg.Height

	m.modeList.SetWidth(width - 4)
	m.targetInput.Width = width - 10
	m.dirInput.Width = width - 10
	m.dirList.SetWidth(width - 4)
	m.progressBar.Width = width - 10
}

func countFilesCmd(dirs []string) tea.Cmd {
	return func() tea.Msg {
		walker := scanner.NewFileWalker()
		total, err := walker.CountFiles(dirs)
		if err != nil {
			return errMsg(err)
		}
		return countFilesMsg{total: total}
	}
}

func progressTick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return progressTickMsg(t)
	})
}

func spinnerTick() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
}

func (m *model) startProcessing() tea.Cmd {
	return func() tea.Msg {
		return processCompleteMsg{}
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (m *model) getTargetDirLabel() string {
	if m.mode == internal.ModeDelete {
		return "3. "
	}
	return "4. "
}

func (m *model) getFocusForDirInput() Focus {
	return FocusDirInput
}

func (m *model) renderStats() string {
	var b strings.Builder
	b.WriteString("📊 实时统计：\n\n")
	b.WriteString(fmt.Sprintf("  总文件数：    %d\n", m.totalFiles))
	b.WriteString(fmt.Sprintf("  已处理：      %d / %d\n", m.processed, m.totalFiles))
	b.WriteString(fmt.Sprintf("  新增记录：    %d 个文件\n", m.stats.Added))
	b.WriteString(fmt.Sprintf("  已删除重复：  %d 个文件\n", m.stats.Deleted))
	b.WriteString(fmt.Sprintf("  已移动重复：  %d 个文件\n", m.stats.Moved))
	freedSpace := formatBytes(m.stats.FreedSpace)
	b.WriteString(fmt.Sprintf("  释放空间：    %s\n", freedSpace))
	return b.String()
}

func (m *model) renderFinalStats() string {
	var b strings.Builder
	b.WriteString("📊 最终统计：\n\n")
	b.WriteString(fmt.Sprintf("  • 扫描目录数：    %d 个\n", len(m.scanDirs)))
	b.WriteString(fmt.Sprintf("  • 总文件数：     %d 个\n", m.totalFiles))
	b.WriteString(fmt.Sprintf("  • 新增记录：     %d 个文件\n", m.stats.Added))
	b.WriteString(fmt.Sprintf("  • 重复文件：     %d 个文件\n", m.stats.Deleted+m.stats.Moved))
	b.WriteString(fmt.Sprintf("    ├─ 已删除：    %d 个\n", m.stats.Deleted))
	b.WriteString(fmt.Sprintf("    └─ 已移动：    %d 个\n", m.stats.Moved))

	elapsed := m.stats.EndTime.Sub(m.stats.StartTime)
	b.WriteString(fmt.Sprintf("  • 释放空间：     %s\n", formatBytes(m.stats.FreedSpace)))
	b.WriteString(fmt.Sprintf("  • 总耗时：      %s\n", elapsed.String()))

	return b.String()
}

func (m *model) logProgress() {
	if m.totalFiles == 0 {
		return
	}

	const logInterval = 100
	if m.processed-m.lastLogProcessed < logInterval && m.processed < m.totalFiles {
		return
	}

	percent := float64(m.processed) / float64(m.totalFiles) * 100
	logger.Get().Info().Msgf("处理进度: %d/%d (%.1f%%) - 新增: %d, 删除: %d, 移动: %d",
		m.processed, m.totalFiles, percent, m.stats.Added, m.stats.Deleted, m.stats.Moved)

	m.lastLogProcessed = m.processed
}

func (m *model) logFinalStats() {
	elapsed := m.stats.EndTime.Sub(m.stats.StartTime)

	logger.Get().Info().Msg("========== 处理完成 ==========")
	logger.Get().Info().Msgf("扫描目录数: %d", len(m.scanDirs))
	logger.Get().Info().Msgf("总文件数: %d", m.totalFiles)
	logger.Get().Info().Msgf("新增记录: %d 个文件", m.stats.Added)
	logger.Get().Info().Msgf("重复文件: %d 个文件", m.stats.Deleted+m.stats.Moved)
	logger.Get().Info().Msgf("  - 已删除: %d 个", m.stats.Deleted)
	logger.Get().Info().Msgf("  - 已移动: %d 个", m.stats.Moved)
	logger.Get().Info().Msgf("释放空间: %s", formatBytes(m.stats.FreedSpace))
	logger.Get().Info().Msgf("总耗时: %v", elapsed)
	logger.Get().Info().Msg("============================")
}
