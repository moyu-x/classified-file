# 文件处理器（File Processor）

这个包实现了一个文件处理器，用于对文件进行去重和分类。

## 代码结构

代码被拆分为多个文件，每个文件负责特定的功能：

- `constants.go`: 定义包级别的常量
- `types.go`: 定义类型和结构体
- `processor.go`: 主处理器结构体和初始化函数
- `file_operations.go`: 文件操作相关函数
- `directory_operations.go`: 目录管理相关函数
- `hash.go`: 哈希计算相关函数
- `file_type.go`: 文件类型检测相关函数

## 主要功能

1. **文件去重**：通过计算文件的xxHash哈希值，检测并处理重复文件
2. **文件分类**：根据文件的MIME类型进行分类
3. **目录管理**：每种文件类型创建单独的目录，并按照指定数量（默认1000个）分配子目录
4. **数据库集成**：将文件哈希和路径信息保存到数据库，便于后续查询

## 使用方法

```go
import (
    "github.com/moyu-x/classified-file/internal/database"
    "github.com/moyu-x/classified-file/internal/fileprocessor"
)

// 创建数据库连接
db, err := database.New("path/to/database")
if err != nil {
    // 处理错误
}

// 创建文件处理器
processor, err := fileprocessor.New(
    "path/to/source",      // 源目录
    "path/to/target",      // 目标目录
    db,                    // 数据库连接
    "path/to/duplicates",  // 重复文件目录
)
if err != nil {
    // 处理错误
}

// 统计源目录中的文件数量
if err := processor.CountTotalFiles(); err != nil {
    // 处理错误
}

// 处理所有文件
if err := processor.ProcessFiles(); err != nil {
    // 处理错误
}

// 获取处理统计信息
stats := processor.Stats
```

## 设计原则

1. **依赖注入**：通过构造函数注入依赖，便于测试和灵活配置
2. **错误处理**：详细的错误信息和日志记录
3. **抽象文件系统**：使用afero库提供文件系统抽象，便于测试
4. **可扩展性**：模块化设计，便于添加新功能 