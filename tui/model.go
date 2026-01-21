package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moyu-x/classified-file/internal"
)

type State int

const (
	StateConfig State = iota
	StateCounting
	StateProcessing
	StateComplete
)

type Focus int

const (
	FocusMode Focus = iota
	FocusTargetDir
	FocusDirInput
	FocusDirList
)

type model struct {
	state            State
	focus            Focus
	mode             internal.OperationMode
	targetDir        string
	scanDirs         []string
	totalFiles       int
	processed        int
	lastLogProcessed int
	stats            internal.ProcessStats
	currentFile      string
	modeList         list.Model
	targetInput      textinput.Model
	dirInput         textinput.Model
	dirList          list.Model
	progressBar      progress.Model
	spinner          spinner.Model
	err              error
}

func initialModel() model {
	modeList := list.New([]list.Item{
		modeItem{title: "直接删除重复文件", desc: "删除所有检测到的重复文件"},
		modeItem{title: "移动到指定目录", desc: "将重复文件移动到指定目录"},
	}, list.NewDefaultDelegate(), 0, 2)

	modeList.Title = "选择重复文件处理方式"
	modeList.SetShowStatusBar(false)
	modeList.SetFilteringEnabled(false)
	modeList.Styles.Title = titleStyle
	modeList.Styles.TitleBar = titleStyle

	targetInput := textinput.New()
	targetInput.Placeholder = "请输入移动目标目录（例如：~/Duplicates）"
	targetInput.Prompt = "> "
	targetInput.PromptStyle = focusedPromptStyle
	targetInput.TextStyle = textStyle

	dirInput := textinput.New()
	dirInput.Placeholder = "请输入要扫描的目录（按回车添加）"
	dirInput.Prompt = "> "
	dirInput.PromptStyle = focusedPromptStyle
	dirInput.TextStyle = textStyle

	dirList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 5)
	dirList.Title = "已添加目录列表"
	dirList.SetShowStatusBar(false)
	dirList.SetFilteringEnabled(false)
	dirList.Styles.Title = titleStyle

	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.PercentageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Width(4)

	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    time.Second / 10,
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		state:       StateConfig,
		focus:       FocusMode,
		mode:        internal.ModeDelete,
		scanDirs:    []string{},
		modeList:    modeList,
		targetInput: targetInput,
		dirInput:    dirInput,
		dirList:     dirList,
		progressBar: progressBar,
		spinner:     s,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

type modeItem struct {
	title string
	desc  string
}

func (m modeItem) Title() string       { return m.title }
func (m modeItem) Description() string { return m.desc }
func (m modeItem) FilterValue() string { return m.title }

type dirItem struct {
	path string
}

func (d dirItem) Title() string       { return d.path }
func (d dirItem) Description() string { return "" }
func (d dirItem) FilterValue() string { return d.path }
