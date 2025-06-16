package fileprocessor

import (
	"github.com/moyu-x/classified-file/internal/database"
	"github.com/spf13/afero"
)

// Stats 保存文件处理的统计信息
type Stats struct {
	TotalFiles     int // 源目录中的总文件数
	ProcessedFiles int // 已成功处理的文件数
	Duplicates     int // 检测到的重复文件数
	Errors         int // 处理过程中的错误数
}

// Processor 文件处理器，负责文件的去重和分类
type Processor struct {
	SourceDir     string          // 源目录路径
	TargetDir     string          // 目标目录路径
	DB            *database.DB    // 数据库连接
	FileHashes    map[string]bool // 文件哈希缓存，用于快速查找重复文件
	Stats         Stats           // 处理统计信息
	Fs            afero.Fs        // 文件系统接口，便于测试和抽象
	MaxDirNumbers map[string]int  // 每种文件类型的最大目录编号
}
