# Classified File

一个用于高效处理、去重和按类型整理文件的命令行工具。

## 功能特点

- 读取源目录中的所有文件
- 高效计算每个文件的xxHash哈希值（替代MD5，性能更佳）
- 将哈希存储在SQLite数据库（纯Go实现，无需CGO）和内存缓存中
- 检测重复文件并将其移动到独立目录（便于用户手动检查）
- 按MIME类型对文件进行分类
- 以结构化的目录层次组织文件（每个文件夹1000个文件）
- 输出每个处理文件的哈希值，便于跟踪

## 安装方法

```bash
go install github.com/moyu-x/classified-file@latest
```

或者克隆并构建:

```bash
git clone https://github.com/moyu-x/classified-file.git
cd classified-file
go build
```

## 使用方法

基本用法:

```bash
classified-file run [选项]
```

选项:

```
  -s, --source string      源目录路径 (必需)
  -t, --target string      目标目录路径 (必需)
  -p, --duplicates string  重复文件存放目录 (必需)
  -d, --db string          SQLite数据库文件路径 (默认为 "./file_hashes.db")
  -v, --debug              启用调试模式
  -h, --help               显示帮助信息
```

示例:

```bash
# 处理从 ~/Downloads 到 ~/Organized 的文件，重复文件放在 ~/Duplicates
classified-file run --source ~/Downloads --target ~/Organized --duplicates ~/Duplicates

# 使用特定的数据库文件
classified-file run --source ~/Downloads --target ~/Organized --duplicates ~/Duplicates --db ~/my-files.db

# 启用调试模式
classified-file run --source ~/Downloads --target ~/Organized --duplicates ~/Duplicates --debug
```

## 工作原理

1. 工具扫描源目录中的所有文件
2. 对每个文件计算xxHash哈希值（比MD5更快）
3. 如果哈希值在数据库中已存在，则该文件被视为重复
   - 重复文件会被移动到指定的重复文件目录
4. 唯一的文件会根据其MIME类型进行分类（使用文件头部信息）
5. 文件按以下结构组织在目标目录中:
   - `/目标目录/[mime类型]/[文件夹编号]/文件名`
   - 每个类型文件夹最多包含1000个文件

## 跨平台支持

使用 Task 进行跨平台构建:

```bash
# 默认构建（当前平台）
task build

# 指定目标平台
task build BUILD_OS=windows BUILD_ARCH=amd64
task build BUILD_OS=linux BUILD_ARCH=amd64
task build BUILD_OS=darwin BUILD_ARCH=arm64

# 构建所有支持的平台
task build-all

# 清理构建产物
task clean
```

## 许可证

MIT License