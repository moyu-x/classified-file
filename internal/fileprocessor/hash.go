package fileprocessor

import (
	"fmt"
	"io"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

// calculateHash 计算文件的xxHash哈希值
func (p *Processor) calculateHash(filePath string) (string, error) {
	// 打开文件
	file, err := p.Fs.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建哈希对象并计算哈希值
	h := xxhash.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("计算哈希失败: %w", err)
	}

	// 将哈希值转换为十六进制字符串
	return strconv.FormatUint(h.Sum64(), 16), nil
}
