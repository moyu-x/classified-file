#!/bin/bash

# 项目验证脚本
# 验证构建、测试和 git 忽略配置是否正确

set -e

echo "=========================================="
echo "  项目配置验证"
echo "=========================================="
echo ""

# 1. 检查 Go 版本
echo "1. Go 版本检查"
GO_VERSION=$(go version | awk '{print $3}')
echo "   Go 版本: $GO_VERSION"

if [[ "$GO_VERSION" < "1.25" ]]; then
    echo "   ⚠️  警告: 建议使用 Go 1.25 或更高版本"
else
    echo "   ✅ Go 版本符合要求"
fi
echo ""

# 2. 检查依赖
echo "2. 依赖检查"
if [ -f "go.mod" ]; then
    echo "   ✅ go.mod 存在"
else
    echo "   ❌ go.mod 不存在"
    exit 1
fi

if [ -f "go.sum" ]; then
    echo "   ✅ go.sum 存在"
else
    echo "   ⚠️  go.sum 不存在，建议运行 go mod tidy"
fi
echo ""

# 3. 检查 go-task
echo "3. go-task 检查"
if command -v task &> /dev/null; then
    TASK_VERSION=$(task --version | awk '{print $3}')
    echo "   ✅ go-task 已安装: $TASK_VERSION"
else
    echo "   ⚠️  go-task 未安装"
    echo "   安装命令: curl -sL https://taskfile.dev/install.sh | sh"
fi
echo ""

# 4. 检查 Taskfile.yml
echo "4. Taskfile.yml 检查"
if [ -f "Taskfile.yml" ]; then
    TASK_COUNT=$(task --list | grep "^*" | wc -l)
    echo "   ✅ Taskfile.yml 存在"
    echo "   ✅ 可用任务数: $TASK_COUNT"
else
    echo "   ❌ Taskfile.yml 不存在"
fi
echo ""

# 5. 检查 .gitignore
echo "5. .gitignore 检查"
if [ -f ".gitignore" ]; then
    echo "   ✅ .gitignore 存在"
    
    # 检查 build 目录是否被忽略
    if grep -q "^build/" .gitignore; then
        echo "   ✅ build/ 目录已配置忽略"
    else
        echo "   ⚠️  build/ 目录未配置忽略"
    fi
    
    # 检查其他构建文件
    BUILD_IGNORED=$(grep -c -E "^build/|^bin/|^classified-file|.*\\.exe$|coverage\\.html|coverage\\.out" .gitignore || true)
    echo "   ✅ 构建文件忽略规则数: $BUILD_IGNORED"
else
    echo "   ⚠️  .gitignore 不存在"
fi
echo ""

# 6. 检查测试文件
echo "6. 测试文件检查"
TEST_FILES=$(find . -name "*_test.go" -type f | grep -v ".git" | wc -l)
echo "   ✅ 测试文件数: $TEST_FILES"

if [ "$TEST_FILES" -gt 0 ]; then
    echo "   ✅ 测试文件存在"
else
    echo "   ⚠️  未找到测试文件"
fi
echo ""

# 7. 检查 build 目录状态
echo "7. build 目录检查"
if [ -d "build/" ]; then
    BUILD_FILES=$(ls build/ | wc -l)
    echo "   ✅ build/ 目录存在，包含 $BUILD_FILES 个文件"
    
    # 检查 build 目录是否被 git 忽略
    if git check-ignore -q build/ 2>/dev/null; then
        echo "   ✅ build/ 目录已被 git 忽略"
    else
        echo "   ⚠️  build/ 目录可能被 git 跟踪"
    fi
else
    echo "   ℹ️  build/ 目录不存在（运行 task build 将创建）"
fi
echo ""

# 8. 检查测试
echo "8. 测试检查"
if command -v task &> /dev/null; then
    echo "   运行测试..."
    if go test ./scanner/... ./hasher/... ./database/... >/dev/null 2>&1; then
        echo "   ✅ 所有测试通过"
    else
        echo "   ⚠️  部分测试失败，运行 task test-all 查看详情"
    fi
fi
echo ""

# 9. 检查编译
echo "9. 编译检查"
if command -v task &> /dev/null; then
    if task build >/dev/null 2>&1; then
        if [ -f "build/classified-file" ]; then
            BUILD_SIZE=$(du -h build/classified-file | cut -f1)
            echo "   ✅ 编译成功: build/classified-file ($BUILD_SIZE)"
        else
            echo "   ❌ 编译产物未找到"
        fi
    else
        echo "   ❌ 编译失败"
    fi
fi
echo ""

echo "=========================================="
echo "  验证完成"
echo "=========================================="
echo ""
echo "下一步："
echo "  1. 运行 'task dev' 进行快速开发循环"
echo "  2. 运行 'task test-all' 查看所有测试"
echo "  3. 运行 './build/classified-file run' 启动程序"
echo "  4. 运行 'task clean' 清理构建产物"
echo ""
