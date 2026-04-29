# Release v2.7.2

## 版本概览

`v2.7.2` 是一次稳定性修复版本，重点修复部分真实小程序样本在 HTML 内嵌 JavaScript 反混淆阶段触发 `nil pointer dereference`，导致 `all` 命令整体 panic 退出的问题。

本版本的目标是：**异常脚本不能中断整包解包流程**。当单个 JS 片段无法完成反混淆分析时，Gwxapkg 会保留原始内容继续写出文件，而不是让进程崩溃。

---

## 重点更新

### 1. 修复 JS / HTML 反混淆 panic

- 修复 HTML `<script>` 内嵌 JavaScript 进入反混淆分析时可能触发的 `nil pointer dereference`
- 反混淆入口增加 panic 兜底，异常样本会被标记为 `skipped` 并保留原始内容
- HTML 格式化流程中，即使内嵌脚本分析失败，也会继续输出 HTML 文件

### 2. 强化 AST 边界防护

- `VariableStatement`、`Binding`、`Identifier`、`ArrayLiteral` 等节点增加空值检查
- AST 遍历逻辑支持 typed nil 节点识别，避免反射遍历时再次触发 panic
- 构建 bootstrap 片段、去重 statement、截取节点源码等辅助路径均增加防御性判断

### 3. 回归测试覆盖

- 新增 formatter 回归测试，覆盖：
  - JavaScript 分析阶段 panic
  - HTML 内嵌 `<script>` 分析阶段 panic
  - 不完整 AST 变量声明
  - typed nil AST 节点遍历
- 已通过：

```bash
go test ./internal/formatter
go test ./...
GOOS=windows GOARCH=amd64 go build -o /tmp/gwxapkg-windows-amd64.exe .
```

---

## 影响范围

- 受影响命令：`all`、默认解包命令、涉及 HTML / JS 格式化与反混淆的流程
- 修复后行为：单个异常 JS 片段最多跳过反混淆，不会导致整次解包失败
- 兼容性：不改变命令行参数，不改变输出目录结构

---

## 下载说明

| 文件 | 适用平台 |
|------|---------|
| `gwxapkg-windows-amd64.exe` | Windows 64 位 |
| `gwxapkg-linux-amd64` | Linux 64 位 |
| `gwxapkg-darwin-amd64` | macOS Intel |
| `gwxapkg-darwin-arm64` | macOS Apple Silicon |

## 使用方法

```bash
# Windows (PowerShell)
.\gwxapkg-windows-amd64.exe scan

# Linux / macOS
chmod +x gwxapkg-linux-amd64
./gwxapkg-linux-amd64 scan
```
