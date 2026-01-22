# Classified File

一个高效的文件分类和去重命令行工具。

## 功能特点

- 高效计算每个文件的 xxHash 哈希值（比 MD5 快 10 倍以上）
- 使用 GORM 框架操作 SQLite 数据库（类型安全的 ORM）
- 检测重复文件并支持两种处理模式：
  - 直接删除重复文件
  - 移动到指定目录
- 支持并发处理，提升性能
- 支持遍历隐藏文件
- 每个文件都显示详细处理日志
- 支持移动模式下的文件名冲突自动重命名

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

### 基本用法

```bash
# 删除重复文件（默认输出每个文件的详细信息）
classified-file ~/Downloads ~/Documents

# 移动重复文件到指定目录
classified-file ~/Downloads --mode move --target-dir ~/Duplicates

# 预览操作（不实际修改文件）
classified-file ~/Downloads --dry-run

# 显示哈希值（verbose 模式）
classified-file ~/Downloads --verbose
```

### 命令行参数

**位置参数:**
- `<directories...>` - 要扫描的目录路径（至少一个）

**选项:**
- `--mode, -m` - 操作模式 (delete|move) [默认: delete]
- `--target-dir, -t` - 移动模式的目标目录 [默认: ""]
- `--db` - 数据库路径 [默认: ~/.classified-file/hashes.db]
- `--workers, -w` - 并发 worker 数量 [默认: 6]
- `--log-level` - 日志级别 [默认: info]
- `--verbose, -v` - 显示哈希值（默认显示文件详情）
- `--dry-run` - 预览模式，不实际修改文件

### 输出说明

**默认输出**（每个文件都会显示详细信息）：
```
[1/1523] 新增记录: /path/to/file.jpg (2.3 MB)
[2/1523] 发现重复: /path/to/duplicate.jpg (2.3 MB, 已删除)
[3/1523] 发现重复: /path/to/file2.jpg (2.3 MB, 已移动到 ~/Duplicates)
```

**--verbose 输出**（额外显示哈希值）：
```
[1/1523] 新增记录: /path/to/file.jpg (2.3 MB, 哈希: a1b2c3d4e5f6g7h8...)
[2/1523] 发现重复: /path/to/duplicate.jpg (2.3 MB, 已删除, 哈希: a1b2c3d4e5f6g7h8...)
```

### 初始化配置

首次使用前，可以初始化配置文件：

```bash
classified-file init
```

这将在 `~/.classified-file/config.yaml` 创建配置文件。

### 配置文件

配置文件位于 `~/.classified-file/config.yaml`，内容示例：

```yaml
# 文件分类去重工具配置文件

database:
  path: "~/.classified-file/hashes.db"

scanner:
  follow_symlinks: false

performance:
  workers: 6

logging:
  level: "info"
  file: ""
```

### 移动模式注意事项

- 文件名格式：移动后的文件名基于 xxHash 值，格式为 `前8位_后8位.扩展名`
- **自动重命名**：如果目标目录中已存在同名文件，会自动添加序号（如 `_1`, `_2` 等）避免冲突
- 例如：`a1b2c3d4_e5f6g7h8.jpg` → `a1b2c3d4_e5f6g7h8_1.jpg` → `a1b2c3d4_e5f6g7h8_2.jpg`
- 使用 `--verbose` 标志可以看到完整的哈希值

## 工作原理

1. **统计阶段**
   - 工具遍历所有指定的目录
   - 统计文件总数（包括隐藏文件）

2. **处理阶段**
   - 对每个文件计算 xxHash 哈希值
   - 在数据库中查找该哈希值：
     - 如果存在，文件被识别为重复文件
       - 删除模式：直接删除文件
       - 移动模式：将文件移动到指定目录
     - 如果不存在，将哈希值和文件信息保存到数据库
   - 每处理一个文件就输出详细日志

3. **完成**
   - 显示处理统计：文件总数、新增记录、删除/移动数量、释放空间等

## 技术栈

- **命令行框架**: [cobra](https://github.com/spf13/cobra)
- **配置管理**: [viper](https://github.com/spf13/viper)
- **哈希算法**: [xxHash](https://github.com/cespare/xxhash/v2)
- **数据库**: [modernc.org/sqlite](https://gitlab.com/cznic/sqlite)（纯 Go 实现）

## 注意事项

- 数据库默认位于 `~/.classified-file/hashes.db`
- 支持处理隐藏文件（以 `.` 开头的文件）
- 首次扫描速度较慢，因为需要计算所有文件的哈希值
- 后续扫描速度会更快，因为已经识别了重复文件
- 移动模式下的文件名会被重命名为 `前缀_后缀.扩展名` 格式，使用哈希值避免文件名冲突

## 构建输出

所有构建产物都输出到 `build/` 目录：

```
build/
├── classified-file              # 当前平台二进制文件
├── classified-file-darwin-arm64   # macOS ARM64
├── classified-file-darwin-amd64   # macOS AMD64
├── classified-file-linux-amd64    # Linux AMD64
├── classified-file-linux-arm64    # Linux ARM64
└── classified-file-windows-amd64.exe  # Windows AMD64
```

**注意**:
- `build/` 目录已被添加到 `.gitignore`，不会提交到版本控制
- 每次运行 `task build` 都会覆盖旧文件
- 运行 `task clean` 会删除 `build/` 目录
- 可运行 `task build-all` 交叉编译所有平台

## 开发和测试

### 安装 go-task

首先需要安装 go-task：

```bash
# macOS/Linux
curl -sL https://taskfile.dev/install.sh | sh

# 或使用 brew (macOS)
brew install go-task/tap/go-task
brew install go-task

# 验证安装
task --version
```

### 使用 Task 命令

项目使用 go-task 管理所有开发和构建任务：

```bash
# 查看所有可用任务
task

# 编译程序（输出到 build/ 目录）
task build

# 为所有平台交叉编译（输出到 build/ 目录）
task build-all

# 运行核心模块测试
task test

# 运行所有测试
task test-all

# 生成测试覆盖率报告（HTML）
task test-coverage

# 查看覆盖率统计（命令行）
task test-coverage-cmd

# 运行竞态条件检测
task test-race

# 启动程序（使用 build/ 目录中的程序）
task run

# 初始化配置文件
task init

# 安装到本地
task install-local

# 下载依赖
task deps

# 更新依赖到最新版本
task deps-update

# 格式化代码
task fmt

# 检查代码格式
task fmt-check

# 运行 go vet
task vet

# 清理构建产物（包括 build/ 目录）
task clean

# 快速开发循环：编译 + 测试
task dev

# 完整 CI 流程：格式检查 + vet + 测试
task ci
```

### 开发工作流

```bash
# 1. 修改代码
# 2. 运行测试
task test

# 3. 编译程序
task build

# 4. 运行程序
task run

# 5. 清理
task clean
```

### 测试覆盖范围

- **scanner 模块** (93.3%): 文件遍历、计数、符号链接处理
- **hasher 模块** (95.2%): 哈希计算、并发处理、大文件处理
- **database 模块** (78.7%): 数据库 CRUD、缓存、持久化
- **deduplicator 模块**: 去重逻辑、文件移动、冲突处理

### 测试安全

测试在临时目录中运行，涉及文件删除和移动操作时使用隔离环境，确保不影响生产数据。

## 许可证

MIT License
