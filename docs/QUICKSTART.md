# 快速开始

## 前置要求

- Go 1.25.1 或更高版本
- go-task（可选，用于便捷的构建和测试）

## 安装

### 1. 克隆仓库

```bash
git clone https://github.com/moyu-x/classified-file.git
cd classified-file
```

### 2. 下载依赖

```bash
# 使用 go-task
task deps

# 或手动
go mod download
go mod tidy
```

### 3. 编译程序

```bash
# 使用 go-task
task build

# 或手动
mkdir build
go build -o build/classified-file main.go
```

### 4. 初始化配置（可选）

```bash
./build/classified-file init
```

## 使用方法

```bash
# 删除重复文件（默认操作）
./build/classified-file ~/Downloads

# 移动重复文件到指定目录
./build/classified-file ~/Documents --mode move --target-dir ~/Duplicates

# 预览操作（不实际修改文件）
./build/classified-file ~/Downloads --dry-run

# 显示详细哈希值
./build/classified-file ~/Downloads --verbose

# 查看帮助
./build/classified-file --help
```

## 测试运行

```bash
# 运行所有测试
task test-all

# 运行核心模块测试
task test

# 生成覆盖率报告
task test-coverage

# 运行带详细输出的测试
go test ./... -v
```

## 开发

```bash
# 运行测试
task test

# 快速开发循环
task dev

# 运行 CI 检查
task ci
```

## 目录结构

```
classified-file/
├── build/              # 构建产物（不提交）
│   ├── classified-file
│   ├── classified-file-darwin-arm64
│   ├── classified-file-darwin-amd64
│   ├── classified-file-linux-amd64
│   ├── classified-file-linux-arm64
│   └── classified-file-windows-amd64.exe
├── cmd/                # 命令行框架
├── config/             # 配置管理
├── scanner/            # 文件遍历
├── hasher/             # 哈希计算
├── database/           # 数据库操作
├── deduplicator/       # 去重逻辑
├── logger/             # 日志管理
├── internal/           # 内部类型和常量
├── Taskfile.yml        # go-task 配置
├── .gitignore          # Git 忽略规则
├── go.mod             # Go 模块
└── README.md          # 项目文档
```

## 故障排除

### 编译错误

```bash
# 清理后重新编译
task clean
task deps
task build
```

### 测试失败

```bash
# 运行所有测试
task test-all

# 运行带详细输出的测试
go test ./... -v
```
