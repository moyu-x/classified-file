package fileprocessor

import (
	"fmt"
	"io"

	"github.com/h2non/filetype"
)

// determineFileType 确定文件的MIME类型
// 读取文件头部并使用filetype库进行类型检测
func (p *Processor) determineFileType(filePath string) (string, error) {
	// 读取文件头部用于类型检测
	head, err := p.readFileHeader(filePath, FileHeaderSize)
	if err != nil {
		return "", fmt.Errorf("读取文件头部失败: %w", err)
	}

	// 检测文件类型
	kind, err := filetype.Match(head)
	if err != nil {
		return "", fmt.Errorf("检测文件类型失败: %w", err)
	}

	// 确定文件类型分类
	if kind == filetype.Unknown {
		return UnknownFileType, nil
	}

	return kind.MIME.Type, nil
}

// readFileHeader 读取文件头部
// 读取文件的前size个字节，用于文件类型检测
func (p *Processor) readFileHeader(filePath string, size int) ([]byte, error) {
	// 打开文件
	file, err := p.Fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取文件头部
	head := make([]byte, size)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取文件头部失败: %w", err)
	}

	// 返回实际读取的字节
	return head[:n], nil
}
