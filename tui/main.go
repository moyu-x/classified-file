package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/moyu-x/classified-file/logger"
)

type Config struct {
	DatabasePath string
}

var cfg *Config

type teaModel struct {
	m *model
}

func (tm teaModel) Init() tea.Cmd {
	return nil
}

func (tm teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return tm.m.Update(msg)
}

func (tm teaModel) View() string {
	return tm.m.View()
}

func Run(config *Config) error {
	cfg = config

	logger.Get().Info().Msg("启动 TUI 界面")

	m := initialModel()
	p := tea.NewProgram(teaModel{m: &m}, tea.WithAltScreen())

	_, err := p.Run()
	if err != nil {
		logger.Get().Error().Err(err).Msg("TUI 运行错误")
	} else {
		logger.Get().Info().Msg("TUI 正常退出")
	}

	return err
}
