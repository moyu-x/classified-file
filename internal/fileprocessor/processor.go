package fileprocessor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moyu-x/classified-file/internal/database"
	"github.com/moyu-x/classified-file/internal/logger"
	"github.com/spf13/afero"
)

// New 创建新的文件处理器
// sourceDir: 源文件目录
// targetDir: 目标文件目录
// db: 数据库连接
// duplicatesDir: 重复文件存放目录
func New(sourceDir, targetDir string, db *database.DB, duplicatesDir string) (*Processor, error) {
	// 加载已有哈希值，用于重复文件检测
	hashes, err := db.LoadHashes()
	if err != nil {
		return nil, fmt.Errorf("加载哈希值失败: %w", err)
	}

	// 验证必要的参数
	if duplicatesDir == "" {
		return nil, fmt.Errorf("重复文件目录不能为空")
	}

	// 创建处理器实例
	processor := &Processor{
		SourceDir:     sourceDir,
		TargetDir:     targetDir,
		DuplicatesDir: duplicatesDir,
		DB:            db,
		FileHashes:    hashes,
		Fs:            afero.NewOsFs(),
		MaxDirNumbers: make(map[string]int),
		// Stats 结构体默认值为零值，无需显式初始化
	}

	// 初始化每种文件类型的最大目录编号
	if err := processor.initMaxDirNumbers(); err != nil {
		return nil, fmt.Errorf("初始化目录编号失败: %w", err)
	}

	return processor, nil
}

// CountTotalFiles 统计源目录中的文件总数
// 遍历源目录，计算非目录文件的数量
func (p *Processor) CountTotalFiles() error {
	count := 0

	// 使用 afero.Walk 遍历目录树
	err := afero.Walk(p.Fs, p.SourceDir, func(path string, info os.FileInfo, err error) error {
		// 处理遍历错误
		if err != nil {
			logger.Debug().Err(err).Str("path", path).Msg("访问路径出错")
			return nil // 跳过错误，继续遍历
		}

		// 只计算文件，不计算目录
		if !info.IsDir() {
			count++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("统计文件数量失败: %w", err)
	}

	// 更新统计信息
	p.Stats.TotalFiles = count
	return nil
}

// ProcessFiles 处理源目录中的所有文件
// 遍历源目录中的所有文件，对每个文件进行处理
func (p *Processor) ProcessFiles() error {
	// 确保重复文件目录存在
	if err := p.Fs.MkdirAll(p.DuplicatesDir, 0755); err != nil {
		return fmt.Errorf("创建重复文件目录失败: %w", err)
	}

	// 记录每种文件类型的最大目录编号
	p.logMaxDirNumbers()

	// 遍历源目录中的所有文件
	return afero.Walk(p.Fs, p.SourceDir, func(path string, info os.FileInfo, err error) error {
		// 处理遍历错误
		if err != nil {
			logger.Debug().Err(err).Str("path", path).Msg("访问路径出错")
			return nil // 跳过错误，继续遍历
		}

		// 跳过目录，只处理文件
		if info.IsDir() {
			return nil
		}

		// 处理单个文件
		if err := p.ProcessFile(path); err != nil {
			// 记录错误并继续处理其他文件
			p.Stats.Errors++
			logger.Error().Err(err).Str("file", path).Msg("处理文件失败")
		} else {
			// 更新处理进度
			p.Stats.ProcessedFiles++

			// 每处理10个文件或处理完所有文件时，输出进度信息
			if p.Stats.ProcessedFiles%10 == 0 || p.Stats.ProcessedFiles == p.Stats.TotalFiles {
				logger.Progress(p.Stats.ProcessedFiles, p.Stats.TotalFiles, "处理进度")
			}
		}
		return nil
	})
}

// logMaxDirNumbers 记录每种文件类型的最大目录编号
func (p *Processor) logMaxDirNumbers() {
	if len(p.MaxDirNumbers) == 0 {
		logger.Info().Msg("没有找到现有的文件类型目录")
		return
	}

	logger.Info().Msg("当前文件类型的最大目录编号:")
	for fileType, maxNum := range p.MaxDirNumbers {
		logger.Info().
			Str("fileType", fileType).
			Int("maxDirNumber", maxNum).
			Msg("文件类型目录编号")
	}
}

// ProcessFile 处理单个文件
// 计算文件哈希，检查重复，分类并移动到目标位置
func (p *Processor) ProcessFile(filePath string) error {
	// 计算文件哈希值
	hashStr, err := p.calculateHash(filePath)
	if err != nil {
		return fmt.Errorf("计算文件哈希失败: %w", err)
	}

	// 输出文件的哈希值
	logger.Info().
		Str("file", filepath.Base(filePath)).
		Str("hash", hashStr).
		Msg("文件哈希值")

	// 检查是否为重复文件
	if p.FileHashes[hashStr] {
		return p.handleDuplicateFile(filePath, hashStr)
	}

	// 处理新文件（非重复文件）
	return p.handleNewFile(filePath, hashStr)
}
