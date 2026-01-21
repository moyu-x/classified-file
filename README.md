# Classified File

一个用于文件分类去重的命令行工具，提供交互式 TUI 界面。

## 功能特点

- 交互式 TUI 界面，易于使用
- 高效计算每个文件的 xxHash 哈希值（比 MD5 快 10 倍以上）
- 使用 GORM 框架操作 SQLite 数据库（类型安全的 ORM）
- 检测重复文件并支持两种处理模式：
  - 直接删除重复文件
  - 移动到指定目录
- 实时显示处理进度和统计信息
- 支持遍历隐藏文件

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

### 初始化配置

首次使用前，建议先初始化配置文件：

```bash
classified-file init
```

这将在 `~/.classified-file/config.yaml` 创建配置文件。

### 启动 TUI 界面

```bash
classified-file run
```

### TUI 界面操作

1. **选择操作模式**
   - 使用方向键选择：直接删除重复文件 或 移动到指定目录
   - 按 Enter 确认

2. **输入移动目标目录**（仅在移动模式下显示）
   - 输入重复文件要移动到的目录
   - 例如：`~/Duplicates`
   - 按 Enter 确认

3. **添加扫描目录**
   - 在"输入要扫描的目录"输入框中输入目录路径
   - 按 Enter 添加到目录列表
   - 可以添加多个目录

4. **开始处理**
   - 按 Tab 键将焦点移到已添加目录列表
   - 按 Enter 开始扫描和处理

5. **其他操作**
   - Tab 键：在各个输入框之间切换焦点
   - Delete/Backspace 键：删除已添加的目录
   - Ctrl+C：退出程序

## 配置文件

配置文件位于 `~/.classified-file/config.yaml`，内容示例：

```yaml
# 文件分类去重工具配置文件

database:
  path: "~/.classified-file/hashes.db"

scanner:
  follow_symlinks: false

performance:
  workers: 6

ui:
  default_mode: "delete"
  default_target_dir: ""

logging:
  level: "info"
  file: ""
```

## 工作原理

1. **扫描阶段**
   - 工具遍历所有添加的目录
   - 统计文件总数（包括隐藏文件）

2. **处理阶段**
   - 对每个文件计算 xxHash 哈希值
   - 在数据库中查找该哈希值：
     - 如果存在，文件被识别为重复文件
       - 删除模式：直接删除文件
       - 移动模式：将文件移动到指定目录
     - 如果不存在，将哈希值和文件信息保存到数据库

3. **完成**
   - 显示处理统计：文件总数、新增记录、删除/移动数量、释放空间等

## 技术栈

- **命令行框架**: [cobra](https://github.com/spf13/cobra)
- **配置管理**: [viper](https://github.com/spf13/viper)
- **TUI 框架**: [bubbletea](https://github.com/charmbracelet/bubbletea) + [bubbles](https://github.com/charmbracelet/bubbles)
- **哈希算法**: [xxHash](https://github.com/cespare/xxhash/v2)
- **数据库**: [modernc.org/sqlite](https://gitlab.com/cznic/sqlite)（纯 Go 实现）

## 注意事项

- 移动模式下的文件名会被重命名为 `前缀_后缀.扩展名` 格式，使用哈希值避免文件名冲突
- 数据库默认位于 `~/.classified-file/hashes.db`
- 支持处理隐藏文件（以 `.` 开头的文件）
- 首次扫描速度较慢，因为需要计算所有文件的哈希值
- 后续扫描速度会更快，因为已经识别了重复文件

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

### 构建输出

所有构建产物都会输出到 `build/` 目录：

```
build/
├── classified-file              # 当前平台二进制
├── classified-file-darwin-arm64
├── classified-file-darwin-amd64
├── classified-file-linux-amd64
├── classified-file-linux-arm64
└── classified-file-windows-amd64.exe
```

**注意**: `build/` 目录已被添加到 `.gitignore`，不会提交到版本控制。

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

### 测试安全

测试在临时目录中运行，涉及文件删除和移动操作时使用隔离环境，确保不影响生产数据。

## 许可证

MIT License
