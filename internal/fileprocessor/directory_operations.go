package fileprocessor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/moyu-x/classified-file/internal/logger"
	"github.com/spf13/afero"
)

// initMaxDirNumbers 初始化每种文件类型的最大目录编号
func (p *Processor) initMaxDirNumbers() error {
	// 检查目标目录是否存在
	exists, err := afero.DirExists(p.Fs, p.TargetDir)
	if err != nil {
		return fmt.Errorf("检查目标目录失败: %w", err)
	}

	// 如果目标目录不存在，则无需初始化
	if !exists {
		return nil
	}

	// 读取目标目录下的所有子目录（文件类型目录）
	typeDirs, err := afero.ReadDir(p.Fs, p.TargetDir)
	if err != nil {
		return fmt.Errorf("读取目标目录失败: %w", err)
	}

	// 遍历每个文件类型目录
	for _, typeDir := range typeDirs {
		// 跳过非目录项
		if !typeDir.IsDir() {
			continue
		}

		fileType := typeDir.Name()
		typeDirPath := filepath.Join(p.TargetDir, fileType)

		// 读取文件类型目录下的所有子目录（序号目录）
		numDirs, err := afero.ReadDir(p.Fs, typeDirPath)
		if err != nil {
			logger.Debug().Err(err).Str("path", typeDirPath).Msg("读取文件类型目录失败")
			continue
		}

		// 找出最大的目录编号
		maxNum := -1
		for _, numDir := range numDirs {
			// 跳过非目录项
			if !numDir.IsDir() {
				continue
			}

			// 尝试将目录名转换为整数
			num, err := strconv.Atoi(numDir.Name())
			if err != nil {
				// 忽略非数字目录名
				continue
			}

			// 更新最大编号
			if num > maxNum {
				maxNum = num
			}
		}

		// 记录该文件类型的最大目录编号
		p.MaxDirNumbers[fileType] = maxNum
		logger.Debug().
			Str("fileType", fileType).
			Int("maxDirNumber", maxNum).
			Msg("初始化文件类型的最大目录编号")
	}

	return nil
}

// getFileCount 获取特定类型的文件计数
// 先从数据库获取，如果失败则通过文件系统计数
func (p *Processor) getFileCount(fileType string) (int, error) {
	// 先尝试从数据库获取计数
	count, err := p.DB.CountFilesByType(fileType)
	if err == nil {
		// 如果这是一个新的文件类型，初始化其最大目录编号为-1
		if _, exists := p.MaxDirNumbers[fileType]; !exists {
			p.MaxDirNumbers[fileType] = -1
			logger.Debug().
				Str("fileType", fileType).
				Msg("初始化新文件类型的目录编号")
		}
		return count, nil
	}

	// 数据库查询失败，回退到文件系统计数
	count, err = p.countFilesByTypeInFilesystem(fileType)
	if err != nil {
		return 0, err
	}

	// 如果这是一个新的文件类型，初始化其最大目录编号为-1
	if _, exists := p.MaxDirNumbers[fileType]; !exists {
		p.MaxDirNumbers[fileType] = -1
		logger.Debug().
			Str("fileType", fileType).
			Msg("初始化新文件类型的目录编号")
	}

	return count, nil
}

// countFilesByTypeInFilesystem 通过文件系统计算特定类型的文件数量
func (p *Processor) countFilesByTypeInFilesystem(fileType string) (int, error) {
	// 构建类型目录路径
	typeDir := filepath.Join(p.TargetDir, fileType)

	// 检查目录是否存在
	exists, err := afero.Exists(p.Fs, typeDir)
	if err != nil {
		return 0, fmt.Errorf("检查目录是否存在失败: %w", err)
	}

	// 目录不存在，返回0
	if !exists {
		return 0, nil
	}

	// 获取文件类型目录下的所有子目录
	dirEntries, err := afero.ReadDir(p.Fs, typeDir)
	if err != nil {
		return 0, fmt.Errorf("读取目录失败: %w", err)
	}

	// 找出所有数字命名的子目录
	numDirs := make(map[int]bool)
	maxDirNum := -1
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}

		// 尝试将目录名转换为整数
		dirName := entry.Name()
		num, err := strconv.Atoi(dirName)
		if err != nil {
			// 忽略非数字目录名
			continue
		}

		numDirs[num] = true
		if num > maxDirNum {
			maxDirNum = num
		}
	}

	// 如果没有找到任何子目录，返回0
	if len(numDirs) == 0 {
		return 0, nil
	}

	// 更新最大目录编号
	p.MaxDirNumbers[fileType] = maxDirNum

	// 计算总文件数
	totalFiles := 0

	// 遍历每个子目录并计算文件数量
	for dirNum := 0; dirNum <= maxDirNum; dirNum++ {
		// 检查该编号的目录是否存在
		if !numDirs[dirNum] {
			// 如果目录不存在，假设之前的目录都已满（每个1000个文件）
			// 这样可以保持目录编号的连续性
			logger.Debug().
				Str("fileType", fileType).
				Int("missingDirNumber", dirNum).
				Msg("目录编号不存在，假设之前的目录已满")
			continue
		}

		// 构建子目录路径
		subDirPath := filepath.Join(typeDir, strconv.Itoa(dirNum))

		// 计算该子目录中的文件数量
		fileCount := 0
		err = afero.Walk(p.Fs, subDirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logger.Debug().Err(err).Str("path", path).Msg("访问路径出错")
				return nil // 忽略错误继续
			}
			if !info.IsDir() {
				fileCount++
			}
			return nil
		})

		if err != nil {
			return 0, fmt.Errorf("统计子目录文件数量失败: %w", err)
		}

		// 如果不是最后一个目录，并且文件数量少于FilesPerDirectory
		// 则假设该目录已满，这样可以保持每个目录1000个文件的规则
		if dirNum < maxDirNum && fileCount < FilesPerDirectory {
			logger.Debug().
				Str("fileType", fileType).
				Int("dirNumber", dirNum).
				Int("actualFileCount", fileCount).
				Int("assumedFileCount", FilesPerDirectory).
				Msg("子目录文件数量不足，假设已满")
			fileCount = FilesPerDirectory
		}

		totalFiles += fileCount

		logger.Debug().
			Str("fileType", fileType).
			Int("dirNumber", dirNum).
			Int("fileCount", fileCount).
			Msg("子目录文件数量")
	}

	logger.Debug().
		Str("fileType", fileType).
		Int("totalCount", totalFiles).
		Int("maxDirNumber", maxDirNum).
		Msg("文件类型的总文件数量")

	return totalFiles, nil
}
