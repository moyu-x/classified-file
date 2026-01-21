package hasher

import (
	"github.com/cespare/xxhash/v2"
	"io"
	"os"

	"github.com/moyu-x/classified-file/logger"
)

func CalculateHash(filePath string) (uint64, error) {
	logger.Get().Debug().Msgf("计算文件哈希: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Get().Error().Err(err).Msgf("无法打开文件: %s", filePath)
		return 0, err
	}
	defer file.Close()

	hash := xxhash.New()
	if _, err := io.Copy(hash, file); err != nil {
		logger.Get().Error().Err(err).Msgf("计算哈希失败: %s", filePath)
		return 0, err
	}

	result := hash.Sum64()
	logger.Get().Trace().Msgf("文件哈希计算完成: %s -> %x", filePath, result)
	return result, nil
}
