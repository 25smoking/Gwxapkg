# Release v2.7.3

## 版本概览

`v2.7.3` 是一次源码语义还原增强版本，重点补齐 `semantic` 后处理链路的 AST 级深度重命名、API 调用链、API 审计伪代码，以及 Burp 原始请求到源码 API 的本地关联能力。

本版本默认启用 `-ast-rename=deep` 激进策略：在保证 `exports.xxx`、对象字段、字符串、注释、WXML handler 和 `wx/uni/getApp/require/module/exports` 等公开或全局标识不被改动的前提下，将 high / medium 置信度的局部变量、函数参数和局部函数名写回为更适合审计的语义名。

---

## 重点更新

### 1. AST 深度语义重命名

- 默认策略从 `safe` 调整为 `deep`
- 支持 `-ast-rename=off|report|safe|deep`
- `deep` 模式会写回 high / medium 置信度候选，例如：
  - `params`
  - `requestData`
  - `response`
  - `event`
  - `app`
  - `resolve` / `reject`
- low 置信度候选只进入报告，不写回源码
- 命令行运行时会提示本次 AST 策略、可能改写范围、保护范围和回滚方式

### 2. AST diff / patch / rollback

- 生成 `.gwxapkg/ast_rename_map.json`
- 生成 `.gwxapkg/ast_rename_diff.md`
- 生成 `.gwxapkg/ast_rename.patch`
- 写回前保留 `.gwxapkg/pre_ast_sources`
- 支持：

```bash
gwxapkg semantic -dir=<已解包目录> -ast-rollback=true
```

### 3. API 调用链与伪代码

- `api_map.json` 增加 `call_chains`
- 新增 `.gwxapkg/api_call_chain.json`
- 新增 `.gwxapkg/api_call_chain.md`
- 新增 `.gwxapkg/api_pseudo.md`
- 新增 `.gwxapkg/pseudo_api/*.js`
- 可从页面调用点追踪到本地 helper，再关联到 `controllerName.methodsName`

### 4. Burp 请求到源码 API 关联

新增 `api-link` 子命令：

```bash
gwxapkg api-link -dir=<已解包目录> -burp-file=<raw_request.txt>
cat raw_request.txt | gwxapkg api-link -dir=<已解包目录>
```

输出：

- `.gwxapkg/burp_api_link.json`
- `.gwxapkg/burp_api_link.md`

匹配策略：

- high：请求参数中直接出现 `controllerName/methodsName`
- medium：HTTP method / path / 参数字段与 `api_map` 高重合
- low：仅参数字段或函数名弱匹配

整个流程只解析 Burp 原始包，不发送网络请求，不重放请求。

### 5. 命令行参数增强

以下入口均支持 AST 策略参数：

- `scan`
- `all`
- 默认 `-id/-in` 解包命令
- `semantic -dir`

参数：

```bash
-ast-rename=off|report|safe|deep
-ast-diff=true
-ast-patch=true
```

---

## 验证

已新增并通过以下回归测试：

- AST `off/report/safe/deep` 模式
- high / medium / low 置信度写回规则
- 对象字段、字符串、注释、公开导出名保护
- diff / patch / rollback
- API 调用链生成
- API 伪代码生成
- Burp JSON / query / form 请求解析与 API 关联

已通过：

```bash
go test ./...
go build ./...
```

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
chmod +x gwxapkg-darwin-arm64
./gwxapkg-darwin-arm64 scan

# 对已解包目录重新执行 semantic
./gwxapkg-darwin-arm64 semantic -dir=<已解包目录>

# Burp 请求关联源码 API
cat raw_request.txt | ./gwxapkg-darwin-arm64 api-link -dir=<已解包目录>
```
