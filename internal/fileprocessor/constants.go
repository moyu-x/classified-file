package fileprocessor

// 文件分类的常量定义
const (
	// FilesPerDirectory 每个子目录中存储的最大文件数
	FilesPerDirectory = 1000

	// UnknownFileType 未知文件类型的标识符
	UnknownFileType = "unknown"

	// FileHeaderSize 文件类型检测所需的文件头部大小（字节）
	FileHeaderSize = 261
)
