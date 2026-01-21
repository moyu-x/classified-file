package tui

import (
	"time"

	"github.com/moyu-x/classified-file/internal"
)

type addDirMsg struct {
	path string
}

type removeDirMsg struct {
	index int
}

type countFilesMsg struct {
	total int
}

type progressMsg struct {
	processed   int
	added       int
	deleted     int
	moved       int
	currentFile string
}

type processCompleteMsg struct {
	stats *internal.ProcessStats
}

type errMsg error

type spinnerTickMsg time.Time

type progressTickMsg time.Time
