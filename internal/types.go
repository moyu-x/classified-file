package internal

import "time"

// 操作模式
type OperationMode string

const (
	ModeDelete OperationMode = "delete"
	ModeMove   OperationMode = "move"
)

// 处理统计
type ProcessStats struct {
	TotalProcessed int
	Added          int
	Deleted        int
	Moved          int
	FreedSpace     int64
	StartTime      time.Time
	EndTime        time.Time
}

// 文件记录
type FileRecord struct {
	ID        int64
	Hash      string
	FilePath  string
	FileSize  int64
	CreatedAt int64
}

// 进度更新
type ProgressUpdate struct {
	Processed   int
	Added       int
	Deleted     int
	Moved       int
	CurrentFile string
}
