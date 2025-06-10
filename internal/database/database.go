package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB 数据库连接实例
type DB struct {
	conn *sql.DB
}

// New 创建一个新的数据库连接
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 创建表（如果不存在）
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS file_hashes (
		hash TEXT PRIMARY KEY,
		file_type TEXT,
		file_path TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_file_type ON file_hashes(file_type);
	`

	_, err = conn.Exec(createTableSQL)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建表失败: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.conn.Close()
}

// LoadHashes 从数据库加载所有哈希值
func (db *DB) LoadHashes() (map[string]bool, error) {
	hashes := make(map[string]bool)

	// 查询所有哈希值
	rows, err := db.conn.Query("SELECT hash FROM file_hashes")
	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("读取行数据失败: %w", err)
		}
		hashes[hash] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	return hashes, nil
}

// SaveHash 将文件哈希保存到数据库
func (db *DB) SaveHash(hash, fileType, filePath string) error {
	_, err := db.conn.Exec(
		"INSERT INTO file_hashes (hash, file_type, file_path) VALUES (?, ?, ?)",
		hash, fileType, filePath,
	)
	if err != nil {
		return fmt.Errorf("插入哈希记录失败: %w", err)
	}
	return nil
}

// CountFilesByType 统计特定类型的文件数量
func (db *DB) CountFilesByType(fileType string) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM file_hashes WHERE file_type = ?", fileType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("查询文件类型数量失败: %w", err)
	}
	return count, nil
}
