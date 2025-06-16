package fileprocessor

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/moyu-x/classified-file/internal/logger"
	"github.com/spf13/afero"
)

// handleDuplicateFile 处理重复文件
// 直接删除重复文件
func (p *Processor) handleDuplicateFile(filePath, hashStr string) error {
	// 更新统计信息
	p.Stats.Duplicates++

	// 记录日志
	logger.Debug().
		Str("file", filePath).
		Str("hash", hashStr).
		Msg("删除重复文件")

	// 删除文件
	if err := p.Fs.Remove(filePath); err != nil {
		return fmt.Errorf("删除重复文件失败: %w", err)
	}

	return nil
}

// handleNewFile 处理新文件（非重复文件）
// 确定文件类型，移动到对应的目标目录
func (p *Processor) handleNewFile(filePath, hashStr string) error {
	// 将哈希存入内存缓存
	p.FileHashes[hashStr] = true

	// 确定文件类型
	fileType, err := p.determineFileType(filePath)
	if err != nil {
		return err
	}

	// 获取文件计数并确定目标目录
	fileCount, err := p.getFileCount(fileType)
	if err != nil {
		return fmt.Errorf("获取文件计数失败: %w", err)
	}

	// 确定当前应该使用的目录编号
	// 每1000个文件一个子目录
	dirNum := fileCount / FilesPerDirectory

	// 检查当前子目录中的文件数量
	currentDirFileCount := fileCount % FilesPerDirectory

	// 如果当前子目录已满或者这是该类型的第一个文件，则使用新的目录编号
	if currentDirFileCount == 0 {
		// 如果是新的目录编号，并且大于已知的最大目录编号，则更新最大目录编号
		if dirNum > p.MaxDirNumbers[fileType] {
			p.MaxDirNumbers[fileType] = dirNum
			logger.Debug().
				Str("fileType", fileType).
				Int("newDirNumber", dirNum).
				Int("fileCount", fileCount).
				Msg("创建新的子目录")
		}
	} else {
		// 如果当前子目录未满，则继续使用当前目录编号
		// 确保目录编号不会超过应有的值
		if dirNum > p.MaxDirNumbers[fileType] {
			p.MaxDirNumbers[fileType] = dirNum
		}
		dirNum = p.MaxDirNumbers[fileType]
		logger.Debug().
			Str("fileType", fileType).
			Int("dirNumber", dirNum).
			Int("currentDirFileCount", currentDirFileCount).
			Msg("继续使用当前子目录")
	}

	// 创建目录结构来存储文件
	dirPath := filepath.Join(p.TargetDir, fileType, strconv.Itoa(dirNum))

	// 确保目标目录存在
	if err := p.Fs.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 将文件移动到目标目录
	destPath := filepath.Join(dirPath, filepath.Base(filePath))
	if err := p.moveFileWithRename(filePath, destPath); err != nil {
		return fmt.Errorf("移动文件失败: %w", err)
	}

	// 将哈希保存到数据库
	if err := p.DB.SaveHash(hashStr, fileType, destPath); err != nil {
		return fmt.Errorf("保存哈希到数据库失败: %w", err)
	}

	// 记录日志
	logger.Debug().
		Str("source", filePath).
		Str("destination", destPath).
		Str("type", fileType).
		Str("hash", hashStr).
		Int("dirNumber", dirNum).
		Int("fileCount", fileCount+1).
		Msg("文件处理完成")

	return nil
}

// moveFile 使用 rename 操作将文件从源路径移动到目标路径
func (p *Processor) moveFile(src, dst string) error {
	// afero 的 Rename 方法在底层对应 os.Rename
	if err := p.Fs.Rename(src, dst); err != nil {
		// 如果 Rename 失败（可能是跨卷移动），尝试复制后删除
		logger.Debug().
			Err(err).
			Str("source", src).
			Str("destination", dst).
			Msg("直接重命名失败，尝试复制后删除")

		// 打开源文件
		sourceFile, err := p.Fs.Open(src)
		if err != nil {
			return fmt.Errorf("打开源文件失败: %w", err)
		}
		defer sourceFile.Close()

		// 创建目标文件
		destFile, err := p.Fs.Create(dst)
		if err != nil {
			return fmt.Errorf("创建目标文件失败: %w", err)
		}
		defer destFile.Close()

		// 复制文件内容
		if _, err = io.Copy(destFile, sourceFile); err != nil {
			return fmt.Errorf("复制文件内容失败: %w", err)
		}

		// 删除原文件
		if err := p.Fs.Remove(src); err != nil {
			return fmt.Errorf("删除原文件失败: %w", err)
		}
	}
	return nil
}

// moveFileWithRename 将文件从源路径移动到目标路径，如果目标文件已存在则自动重命名
func (p *Processor) moveFileWithRename(src, dst string) error {
	// 检查目标文件是否存在
	exists, err := p.fileExists(dst)
	if err != nil {
		return fmt.Errorf("检查文件是否存在失败: %w", err)
	}

	// 如果文件不存在，直接移动
	if !exists {
		return p.moveFile(src, dst)
	}

	// 文件已存在，需要重命名
	// 分离文件名和扩展名
	ext := filepath.Ext(dst)
	baseName := strings.TrimSuffix(dst, ext)

	// 使用时间戳作为后缀，确保文件名唯一
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	newDst := fmt.Sprintf("%s_%d%s", baseName, timestamp, ext)

	// 记录重命名日志
	logger.Debug().
		Str("original_path", dst).
		Str("new_path", newDst).
		Msg("文件名冲突，自动重命名")

	// 移动到新的目标路径
	return p.moveFile(src, newDst)
}

// fileExists 检查文件是否存在
func (p *Processor) fileExists(path string) (bool, error) {
	return afero.Exists(p.Fs, path)
}
