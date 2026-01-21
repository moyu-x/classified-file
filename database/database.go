package database

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
)

type FileRecord struct {
	ID        int64     `gorm:"primaryKey"`
	Hash      string    `gorm:"uniqueIndex;not null"`
	FilePath  string    `gorm:"not null"`
	FileSize  int64     `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (FileRecord) TableName() string {
	return "file_hashes"
}

type Database struct {
	db    *gorm.DB
	cache map[string]bool
	mu    sync.RWMutex
}

func NewDatabase(dbPath string) (*Database, error) {
	expandedPath, err := expandPath(dbPath)
	if err != nil {
		logger.Get().Error().Err(err).Msg("扩展数据库路径失败")
		return nil, err
	}

	logger.Get().Info().Msgf("初始化数据库，路径: %s", expandedPath)

	if err := os.MkdirAll(filepath.Dir(expandedPath), 0755); err != nil {
		logger.Get().Error().Err(err).Msgf("创建数据库目录失败: %s", filepath.Dir(expandedPath))
		return nil, err
	}

	dsn := expandedPath + "?_pragma=journal_mode(WAL)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Get().Error().Err(err).Msg("打开数据库连接失败")
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Get().Error().Err(err).Msg("获取数据库连接失败")
		return nil, err
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := createSchema(db); err != nil {
		logger.Get().Error().Err(err).Msg("创建数据库表失败")
		return nil, err
	}

	logger.Get().Info().Msg("数据库初始化完成")
	return &Database{
		db:    db,
		cache: make(map[string]bool),
		mu:    sync.RWMutex{},
	}, nil
}

func expandPath(path string) (string, error) {
	if len(path) >= 2 && path[0] == '~' && (path[1] == '/' || path[1] == '\\') {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func createSchema(db *gorm.DB) error {
	return db.AutoMigrate(&FileRecord{})
}

func (d *Database) Exists(hash string) (bool, error) {
	d.mu.RLock()
	exists, ok := d.cache[hash]
	d.mu.RUnlock()

	if ok {
		logger.Get().Trace().Msgf("从缓存中查询哈希: %s -> %v", hash, exists)
		return exists, nil
	}

	var count int64
	if err := d.db.Model(&FileRecord{}).Where("hash = ?", hash).Count(&count).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("查询哈希失败: %s", hash)
		return false, err
	}

	exists = count > 0

	d.mu.Lock()
	d.cache[hash] = exists
	d.mu.Unlock()

	logger.Get().Trace().Msgf("从数据库中查询哈希: %s -> %v", hash, exists)
	return exists, nil
}

func (d *Database) Insert(record *internal.FileRecord) error {
	gormRecord := &FileRecord{
		Hash:      record.Hash,
		FilePath:  record.FilePath,
		FileSize:  record.FileSize,
		CreatedAt: time.Unix(record.CreatedAt, 0),
	}

	if err := d.db.Create(gormRecord).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("插入记录失败: %s", record.FilePath)
		return err
	}

	d.mu.Lock()
	d.cache[record.Hash] = true
	d.mu.Unlock()

	logger.Get().Debug().Msgf("插入记录成功: %s (大小: %d bytes)", record.FilePath, record.FileSize)
	return nil
}

func (d *Database) Close() error {
	logger.Get().Info().Msg("关闭数据库连接")
	sqlDB, err := d.db.DB()
	if err != nil {
		logger.Get().Error().Err(err).Msg("获取数据库连接失败")
		return err
	}
	return sqlDB.Close()
}
