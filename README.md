# Gwxapkg

<div align="center">

![Version](https://img.shields.io/badge/version-2.7.3-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)
![Build](https://img.shields.io/badge/build-passing-brightgreen.svg)

**[中文](README.md) | [English](README_EN.md) | [日本語](README_JA.md)**

一款基于Go实现的微信小程序 `.wxapkg` 解包工具，支持自动扫描、解密、反编译和安全分析。

</div>

---

## ⚠️ 重要法律与使用声明

**本工具仅限于在合法、正当且已获得充分授权的前提下，用于安全研究、逆向分析、兼容性验证、学习交流以及自有或受托资产的技术审计。**

**任何用户在使用本工具前，均应自行确认其对目标小程序、相关账号、设备、数据、网络环境、业务系统及其他关联资源拥有明确、持续且可证明的合法授权。若无法确认授权范围或授权有效性，请立即停止使用。**

### 禁止用途

**严禁将本工具用于任何未经授权或可能违反法律法规、平台规则、合同义务、知识产权规则或隐私保护要求的场景，包括但不限于：**

- **未经许可解包、分析、提取、复制、传播第三方小程序代码、资源或数据**
- **绕过平台保护机制、风控措施、访问限制或其他技术控制**
- **批量采集、数据抓取、隐私信息提取、接口滥用或自动化攻击测试**
- **用于商业盗用、恶意仿制、恶意传播、灰黑产活动或其他不正当用途**
- **实施任何可能导致他人权益受损、平台处罚、业务中断、数据泄露或合规风险的行为**

### 风险提示

**使用本工具可能涉及并产生包括但不限于以下风险：法律责任、行政处罚、民事赔偿、刑事风险、知识产权争议、隐私侵权争议、账号封禁、服务中断、数据泄露、生产事故及第三方索赔。**

**用户应自行评估并承担因其使用、误用或滥用本工具所产生的一切风险与后果。**

### 责任限制

**在适用法律允许的范围内，本项目及其作者、贡献者、维护者不对因使用、无法使用、误用或滥用本工具而导致的任何直接、间接、附带、特殊、惩罚性或后果性损失承担责任，包括但不限于数据丢失、信息泄露、隐私侵权、知识产权纠纷、平台处罚、账号封禁、系统故障、业务损失、经济损失、行政责任或刑事责任。**

**本工具按“现状”提供，不附带任何明示或默示保证，包括但不限于适销性、特定用途适用性、稳定性、准确性、完整性、持续可用性或适法性保证。**

### 使用即表示同意

**继续下载、安装、运行、分发或使用本工具，即视为你已阅读、理解并同意本声明的全部内容，并承诺仅在合法授权范围内使用本工具。**

---

## ✨ 核心特性

### 🔍 智能解包
- **自动扫描** - 自动检测 macOS/Windows 微信小程序缓存目录
- **自动解密** - 支持加密的 wxapkg 文件自动解密（PC端）  
- **一键解包** - 自动查找并处理指定 AppID 的所有文件
- **分包处理** - 正确处理主包和分包的依赖关系

### 🎨 代码还原
- **完整还原** - wxml/wxss/js/json/wxs 全部支持
- **代码美化** - 自动格式化 JavaScript/CSS/HTML 代码
- **默认反混淆** - JavaScript 默认执行静态还原 + 受控解码，优先展开常见字符串数组、`\xNN`、`\uNNNN`、十六进制字面量
- **源码级语义还原** - 默认启用 AST `deep` 激进策略，把压缩变量/函数名恢复为更适合审计的 `params`、`requestData`、`response`、`event`、`app` 等语义名
- **可追溯 AST 写回** - 生成 `ast_rename_map.json`、`ast_rename_diff.md`、`ast_rename.patch`，并保留写回前源码用于 `semantic -ast-rollback=true` 回滚
- **API 语义视图** - 自动生成 `api_map`、API 调用链、审计伪代码，并支持将 Burp 原始请求关联到源码 API
- **页面路由地图** - 自动生成页面清单、入口页、分包、TabBar、组件依赖、静态/动态跳转边、事件触发线索与页面接口映射
- **目录结构** - 还原微信小程序原始工程目录
- **资源提取** - 图片/音频/视频等资源文件完整提取

### 🛡️ 安全分析 ⭐ NEW
- **智能扫描** - 920 条默认敏感信息检测规则
- **正式分类** - 规则统一归并到云平台、支付、通知协作、监控、安全、SaaS 等正式大类
- **误报过滤** - 黑名单 + 占位符/示例值/掩码值/弱值过滤，多层收敛扫描噪声
- **数据去重** - 自动去除重复数据，精准定位
- **接口提取** - 自动提取 URL / API Endpoint，并可导出 Postman Collection
- **Excel/HTML报告** - 专业多Sheet Excel 与交互式 HTML 报告，包含文件路径和行号
- **风险分级** - 高/中/低风险自动分类
- **混淆标记** - 在报告中单独列出命中的混淆文件及还原状态

### ⚡ 性能优化
- **动态并发** - 根据CPU核心数自动调整并发度
- **缓冲I/O** - 256KB缓冲区，大幅提升文件读写性能  
- **规则预编译** - 启动时编译正则，避免重复开销
- **编译优化** - 使用优化编译标志，减小体积提升速度

---

## 📊 支持的文件类型

| 文件类型 | 支持情况 | 说明 |
|----------|----------|------|
| `.wxml` | ✅ | 页面结构还原 |
| `.wxss` | ✅ | 样式文件还原 |
| `.js` | ✅ | JavaScript 代码还原 + 美化 + 默认反混淆 |
| `.json` | ✅ | 配置文件提取 |
| `.wxs` | ✅ | WXS 脚本还原 |
| 图片/音频/视频 | ✅ | 资源文件完整提取 |

---

## 📥 安装

### 方式一：下载预编译版本（推荐）

前往 [Releases](https://github.com/25smoking/Gwxapkg/releases) 页面下载对应平台的可执行文件。

### 方式二：从源码编译

```bash
# 克隆仓库
git clone https://github.com/25smoking/Gwxapkg.git
cd Gwxapkg

# 编译（优化版本）
go build -ldflags="-s -w" -o gwxapkg .

# 或直接运行
go run . -h
```

**系统要求：** Go 1.21 或更高版本

---

## 🚀 快速开始

### 基本用法

```bash
# 自动扫描并处理指定 AppID 的小程序
./gwxapkg all -id=<AppID>

# 查看所有可用的小程序
./gwxapkg scan

# 查看微信缓存候选路径诊断
./gwxapkg scan --verbose

# 解包单个 wxapkg 文件
./gwxapkg -id=<AppID> -in=<文件路径>

# 对已解包目录独立扫描，并额外导出 Postman Collection
./gwxapkg scan-only -dir=<目录> -format=both -postman

# 重新打包
./gwxapkg repack -in=<目录路径>
```

### 命令参数

| 参数 | 说明 | 默认值 |
|------|------|------|
| `-id` | 小程序 AppID（必填） | - |
| `-in` | 输入文件/目录路径 | - |
| `-out` | 输出目录 | 自动生成 |
| `-restore` | 还原工程目录结构 | true |
| `-pretty` | 美化代码输出 | true |
| `-sensitive` | 启用敏感信息扫描 | true |
| `-postman` | 导出 `api_collection.postman_collection.json` | false |
| `-noClean` | 保留中间临时文件 | false |
| `-save` | 保存解密后的文件 | false |
| `-workspace` | 保留可精确回包的隐藏工作区 | false |
| `--verbose` | 输出微信缓存候选路径诊断（仅 `scan` / `all`） | false |

### 使用示例

```bash
# 示例1: 自动扫描并处理
./gwxapkg all -id=wx3c19e32cb8f31289

# 示例2: 解包并导出 Postman Collection
./gwxapkg all -id=wx3c19e32cb8f31289 -postman

# 示例3: 仅解包单个文件
./gwxapkg -id=wx123456 -in=test.wxapkg -out=./output

# 示例4: 对已解包目录重新扫描
./gwxapkg scan-only -dir=./output/wx123456 -format=both -postman

# 示例5: 重新打包
./gwxapkg repack -in=./source_dir -out=new.wxapkg
```

### 默认输出目录

未指定 `-out` 时，输出目录规则如下：

- 正式编译后的可执行程序：输出到程序所在目录下的 `output/<AppID>`
- `go run .` 或开发环境：输出到当前工作目录下的 `output/<AppID>`
- `scan` 交互模式：默认同样落到 `output/<AppID>`

示例：

```text
/Applications/Gwxapkg/output/wx1234567890abcdef
./output/wx1234567890abcdef
```

### 典型输出结构

```text
output/
└── wx1234567890abcdef/
    ├── app.js
    ├── page-frame.html
    ├── sensitive_report.xlsx
    ├── sensitive_report.html
    ├── api_collection.postman_collection.json
    ├── route_manifest.json
    ├── route_map.md
    ├── route_map.mmd
    └── .gwxapkg/                   # 仅在 -workspace=true 时生成
```

---

## 📁 微信小程序缓存位置

### macOS
```
~/Library/Containers/com.tencent.xinWeChat/Data/Library/Caches/
├── applet/
│   ├── release/
│   └── debug/
└── ...
```

### Windows
```
%USERPROFILE%\Documents\WeChat Files\Applet\
├── wx<appid>/
│   ├── <version>/
│   │   ├── __APP__.wxapkg      # 主包
│   │   └── __SUBCONTEXT__.wxapkg  # 分包
│   └── ...
└── ...
```

---

## 🎯 敏感信息扫描

### 扫描规则（920 条默认规则）

当前默认启用 920 条内置规则。虽然新增规则里大量命名仍然是 `xxx secret` 风格，但程序内部已经不再按字面名称粗暴归类，而是优先按服务域和用途统一整理成正式大类，避免报告里出现大量 `other` 或一股脑堆进 `secret`。

| 分类 | 规则数 | 示例 |
|------|--------|------|
| **第三方 SaaS** | 596 | 各类 `xxx secret`、服务专属 token、Webhook、客户端凭证 |
| **Secret / 密钥** | 67 | 通用 `client_secret`、`app_secret`、`session_secret` |
| **开发与交付** | 40 | GitHub/GitLab/NPM/PyPI/Jenkins/Terraform/JFrog |
| **云平台** | 35 | AWS、阿里云、腾讯云、华为云、Azure、Cloudflare |
| **支付与电商** | 35 | Stripe、PayPal、Square、Razorpay、微信支付、Shopify |
| **通知与协作** | 31 | Slack、Discord、Telegram、钉钉、飞书、Teams、Twilio |
| **监控与告警** | 24 | Datadog、New Relic、Sentry、Grafana、PagerDuty |
| **Token / 令牌** | 16 | JWT、Bearer、OAuth Token、会话令牌 |
| **数据库与连接** | 15 | MySQL、PostgreSQL、MongoDB、Redis、ES、InfluxDB |
| **密码** | 13 | 通用密码、Root/管理员密码、数据库/协议密码 |
| **安全平台** | 13 | Auth0、Shodan、Censys、VirusTotal、AbuseIPDB |
| **私钥与证书** | 11 | RSA/DSA/EC/OpenSSH/PKCS8 私钥、证书 |
| **编码与指纹** | 8 | Hash、UUID、长 Base64、SSH 公钥 |
| **联系信息** | 3 | 手机号、邮箱、身份证 |
| **网络标识** | 3 | IPv4、内网 IP、MAC 地址 |
| **API 密钥** | 3 | 通用 API Key / Access Key |
| **微信生态** | 3 | AppID、CorpID、Secret |
| **URL / API** | 2 | URL、API Endpoint |
| **路径** | 1 | 文件路径、系统路径 |
| **域名** | 1 | 域名地址（带 TLD 校验） |

### 误报控制策略

为了在扩到 920 条规则后仍然尽量可用，当前扫描器默认会做这几层收敛：

- **黑名单过滤**：跳过常见静态资源名、框架 API 片段和明显非敏感内容
- **上下文过滤**：域名类继续做 TLD 校验、JS API 语境过滤、路径长度校验
- **占位符过滤**：自动排除 `your_api_key`、`replace_me`、`changeme`、`placeholder` 这类示例值
- **掩码值过滤**：自动排除 `xxxxxx`、`******`、`<token>` 这类脱敏或占位文本
- **弱值过滤**：对凭证类结果增加最小长度、字符形态和普通词检测，减少明显弱命中


### 扫描与导出行为

- `-sensitive=true` 时生成 `sensitive_report.xlsx` 和 `sensitive_report.html`
- `-postman=true` 时生成 `api_collection.postman_collection.json`
- `-postman` 与 `-sensitive` 解耦，可以单独开启
- `scan-only` 会复用同一套扫描器与 JS 反混淆逻辑
- 无法可靠推断 HTTP 方法时，Postman 中会写入 `UNKNOWN`
- 相对接口路径会原样保留，不会自动拼接 `baseUrl`

## 🧭 页面与路由地图

解包完成后的输出目录现在会默认额外生成：

- `route_manifest.json`：机器可读的页面与路由结构清单
- `route_map.md`：便于人工审阅的页面说明文档
- `route_map.mmd`：Mermaid 图结构，可直接渲染页面跳转关系

当前能力已覆盖：

- 入口页识别
- 主包 / 分包页面清单
- TabBar 页面
- 页面标题与页面文件映射
- `usingComponents` 组件依赖
- `wx.navigateTo` / `redirectTo` / `reLaunch` / `switchTab`
- WXML `<navigator>` 静态跳转
- WXML `bindtap` / `catchtap` 触发元素与 handler 反查
- `data-url` / `data-route` / `data-path` / `data-page` 这类数据目标回填
- 模板字符串、字符串拼接、`dataset.url` 这类动态路由线索
- 页面按钮文案 / 触发事件 / handler 名关联到跳转边
- 页面脚本中的直接接口调用归属
- 页面引用共享服务模块后的间接接口归属
- 页面 handler -> 本地 helper -> 共享 helper -> 最终跳转 的跨文件调用链恢复
- `utils/router.js`、`common/nav.js` 这类共享路由助手识别与归档
- 未在 `app.json` 中声明但脚本中出现 `Page(...)` 的孤页候选


### 报告内容

生成的 Excel / HTML 报告包含以下内容：

- **概览** - 扫描统计、风险分布、分类汇总
- **分类页** - 按实际命中动态生成，例如云平台、支付与电商、通知与协作、开发与交付、第三方 SaaS
- **URL / API** - URL、API Endpoint 与接口上下文
- **数据库与连接** - JDBC、MongoDB、Redis、ES 等连接信息
- **密码 / Secret / Token** - 密码、通用密钥、会话或访问令牌
- **微信生态** - 微信相关配置
- **混淆文件** - 命中的混淆文件、分数、技术点、还原状态

每条数据包含：
- ✅ 内容（Content）
- ✅ 出现次数
- ✅ 文件路径
- ✅ 行号
- ✅ 风险等级

混淆文件额外包含：
- ✅ 状态（`restored` / `partial` / `flagged`）
- ✅ 分数（Score）
- ✅ 命中技术（Techniques）
- ✅ 标签（`[OBFUSCATED] ...`）

### Postman Collection 示例

```json
{
  "info": {
    "name": "wx1234567890abcdef - API Collection"
  },
  "item": [
    {
      "name": "POST /api/user/login",
      "request": {
        "method": "POST",
        "url": {
          "raw": "/api/user/login"
        }
      }
    }
  ]
}
```

### 规则配置说明

- 默认情况下，程序会直接使用内置规则集
- 不会自动写出 `config/rule.yaml`
- 如果你手动放置了 `config/rule.yaml`，则优先使用你的自定义规则覆盖内置规则
- 适合在不改源码的情况下按自己的审计口径裁剪规则

---

## 📈 性能对比（v2.7.3 vs v1.0）

| 指标 | v1.0 | v2.7.3 | 改进 |
|------|------|--------|------|
| **扫描速度** | 基准 | +50-70% | ⬆️⬆️⬆️ 规则预编译 |
| **误报控制** | 基础正则直扫 | 多层过滤收敛 | ✅ 黑名单 + 上下文 + 占位符 + 弱值过滤 |  
| **数据量** | 127,185条 | ~3,000条 | ⬇️⬇️⬇️ 去重+过滤 |
| **输出格式** | JSON | Excel/HTML | ✅ 交互式报告 |
| **并发性能** | 10固定 | CPU*2动态 | ⬆️⬆️ 自适应 |
| **I/O性能** | 直接写入 | 256KB缓冲 | ⬆️⬆️ 减少系统调用 |

---

## 🔄 版本更新

### v2.7.3 (2026-05-12) - 🧠 Semantic 与 AST 深度还原增强

#### 🆕 新增功能
- 新增 AST 变量/函数深度重命名，默认 `-ast-rename=deep`，对 high/medium 置信度的局部变量、函数参数和局部函数名进行源码级语义写回
- 新增 `off/report/safe/deep` 四档 AST 策略，并在命令行运行时提示本次策略、可能改写范围、保护范围和回滚方式
- 新增 `.gwxapkg/ast_rename_map.json`、`.gwxapkg/ast_rename_diff.md`、`.gwxapkg/ast_rename.patch`，保留 `.gwxapkg/pre_ast_sources` 以支持回滚
- 新增 API 调用链产物 `.gwxapkg/api_call_chain.json` / `.gwxapkg/api_call_chain.md`
- 新增 AST API 审计伪代码 `.gwxapkg/api_pseudo.md` 与 `.gwxapkg/pseudo_api/*.js`
- 新增 `api-link` 子命令，支持从 Burp 原始请求文件或 stdin 自动关联到源码 API

#### 🔧 改进
- `scan`、`all`、默认解包命令和 `semantic -dir` 均支持 `-ast-rename`、`-ast-diff`、`-ast-patch`
- AST 写回明确保护 `exports.xxx`、对象字段、字符串、注释、WXML handler 和 `wx/uni/getApp/require/module/exports` 等公开或全局标识
- `semantic_module_map.json` 增加 AST 统计、API 调用链路径和 API 伪代码路径，便于审计追溯

#### ✅ 验证
- 已新增 AST 模式、置信度、diff/patch/rollback、API 调用链、伪代码和 Burp 请求关联回归测试
- 已验证 `go test ./...`、`go build ./...`

### v2.7.2 (2026-04-30) - 🛡️ 稳定性与报告修复

#### 🔧 修复
- 修复 HTML 内嵌 `<script>` 在 JS 反混淆分析阶段可能触发 `nil pointer dereference` 并导致 `all` 命令整体崩溃的问题
- 强化 `goja` AST 遍历与变量声明识别的空节点防护，覆盖不完整 `VariableStatement`、`Binding`、typed nil 节点等边界情况
- 修复反混淆静态解码 `\x22`、`\u0022` 等字符串转义后可能破坏 JS 语法的问题
- 修复 Excel 报告分类名包含非法 sheet 字符时 `.xlsx` 生成失败的问题
- 改进 WXML 还原，跳过 `virtual` 虚拟节点并为动态渲染分支增加兜底参数
- 当单个 JS 片段反混淆分析异常时，会保留原始内容并继续解包，避免异常样本中断整包处理

#### ✅ 验证
- 新增 formatter、reporter、unpack 回归测试，覆盖 JS 分析 panic、HTML 内嵌脚本 panic、异常 AST 节点、Excel sheet 名清洗与 WXML 虚拟节点展开
- 已验证 `go test ./...` 与 Windows amd64 交叉编译

### v2.7.1 (2026-04-21) - 🧭 路由分析增强

#### 🆕 新增功能
- 🔗 **跨页面调用链恢复** - 页面 `handler -> 本地 helper -> 共享 helper -> 最终跳转` 现在会写入 `route_manifest.json`
- 🧭 **共享路由助手识别** - 自动识别 `utils/router.js`、`common/nav.js` 这类共享跳转封装，并统计使用页面、方法与目标线索

#### 🔧 改进
- 📊 **路由报告增强** - `route_map.md` / `route_map.mmd` 会展示调用链与共享路由助手信息
- 🧹 **降级边裁剪** - 当已恢复出真实跳转方法时，自动去掉重复的 `UNKNOWN` 兜底边，减少噪声

### v2.7.0 (2026-04-18) - 💥 大版本功能增强

#### 🆕 新增功能
- 📊 **HTML 交互式报告** - 新增了美观的可视化安全分析 HTML 报告格式，带饼图、快速过滤与分类选项。
- 🔍 **`scan-only` 独立命令** - 支持对纯净的已解包目录直接发起代码/敏感信息审查，不被解包流程束缚。
- 🗂️ **全自动化集成流** - 深度融入 GitHub Actions，支持跨平台 Release 推送及 CI 测试流程。
- 🪟 **Windows AppName 提取** - Gwxapkg 现在能提取出小程序包名称。
- ⚙️ **批量扫描支持扩展** - 支持通过 `-id="wx1,wx2"` 以及 `-id-file=ids.txt` 大规模批量化导出特定程序。
- 📮 **Postman Collection 导出** - `all` / 默认命令 / `scan` / `scan-only` 均支持 `-postman`
- 🧠 **默认 JS 反混淆** - 增加静态还原、AST 识别、受控 `goja` 解码执行
- 🏷️ **混淆文件报告** - Excel / HTML 概览新增混淆文件统计，并提供单独清单
- 🧭 **缓存路径诊断** - `scan --verbose` / `all --verbose` 输出微信缓存候选路径诊断
- 📦 **内置规则优先** - 默认直接使用内置规则，不再自动生成 `rule.yaml`
- 🗃️ **规则分类重整** - 将 920 条规则统一整理为云平台、支付、协作、监控、安全、SaaS 等正式大类
- 🧹 **误报控制增强** - 新增占位符、掩码值、弱值过滤，降低 920 条规则扩容后的噪声

### v2.6.0 (2026-03-17) - 🚀 稳定版本

#### 🆕 新增功能
- 🔧 **稳定性提升** - 修复若干已知问题，提升整体运行稳定性
- 📈 **规则优化** - 持续优化敏感信息扫描规则，减少误报
- 🖥️ **兼容性增强** - 改善 Windows/macOS/Linux 跨平台兼容性
- 📦 **依赖更新** - 更新第三方依赖至最新稳定版本

#### 🐛 修复问题
- 修复部分情况下分包处理异常的问题
- 修复 Excel 报告在特殊字符时的生成错误
- 优化大型小程序的内存占用

---

### v2.5.0 (2025-12-05) - 🎉 重大更新

#### 🆕 新增功能
- ✨ **Excel报告生成** - 专业的多Sheet分类报告，替代简单JSON
- 🎯 **智能误报过滤** - 黑名单+TLD验证+上下文检测，误报率降低85%
- 📊 **数据去重** - 自动去重，数据量减少97%
- 🏷️ **风险分级** - 高/中/低风险自动分类
- 📍 **完整上下文** - 每条数据包含文件路径和行号

#### ⚡ 性能优化
- 🚀 **动态并发** - 根据CPU核心数自动调整worker数量（原固定10→CPU*2）
- 💾 **缓冲I/O** - 256KB缓冲区提升文件读写性能
- 🔧 **规则预编译** - 启动时编译所有正则，避免重复开销
- 📦 **编译优化** - 使用 `-ldflags="-s -w"` 减小体积

#### 🐛 修复问题
- 修复domain规则误匹配文件名（如index.weapp）
- 修复JavaScript API被误识别为域名
- 优化目录合并性能

#### 💡 技术改进
- 新增 `internal/scanner` 模块（types, filter, collector, scanner）
- 新增 `internal/reporter` 模块（Excel报告生成）
- 使用 `excelize/v2` 库生成专业Excel报告  
- 完整的单元测试覆盖

### v1.0.0 (2024-XX-XX)
- 🎉 初始发布
- ✅ 基础解包功能
- ✅ 代码美化
- ✅ 敏感信息扫描（JSON输出）

---

## 🛠️ 技术架构

```
Gwxapkg/
├── cmd/
│   └── root.go           # CLI入口，进度条，报告生成
├── internal/
│   ├── cmd/              # 命令处理，文件解析
│   ├── decrypt/          # AES+XOR解密
│   ├── unpack/           # wxapkg二进制解析
│   ├── restore/          # 工程结构还原
│   ├── formatter/        # 代码美化与 JS 反混淆
│   │   ├── jsformatter.go
│   │   └── deobfuscator.go
│   ├── key/              # 规则管理，预编译
│   ├── scanner/          # ⭐ NEW 扫描引擎
│   │   ├── types.go      # 数据模型
│   │   ├── rule_meta.go  # 规则分类、名称、风险等级
│   │   ├── filter.go     # 误报过滤
│   │   ├── collector.go  # 数据收集和去重
│   │   ├── scanner.go    # 扫描逻辑
│   │   └── api_extractor.go
│   ├── reporter/         # ⭐ NEW 报告生成
│   │   ├── excel.go      # Excel报告
│   │   ├── html.go       # HTML报告
│   │   └── postman.go    # Postman Collection 导出
│   ├── config/           # 配置管理
│   └── ui/               # 终端UI
├── config/
│   └── rule.yaml         # 可选的自定义规则覆盖文件
└── main.go
```

---

## 🤝 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## ❓ 常见问题 (FAQ)

### 1. 为什么双击运行会“闪退”？
**该工具是命令行工具 (CLI)**，不支持直接双击运行。
- **错误做法**：在资源管理器中直接双击 `gwxapkg.exe`。这会导致程序运行完毕或报错后立即关闭窗口，看起来像“闪退”。
- **正确做法**：打开终端（如 CMD、PowerShell 或 Terminal），先 `cd` 到工具所在目录，然后输入命令运行。

### 2. 找不到小程序包？
请确保你已经登录过 PC 版微信并打开过目标小程序。如果依然找不到，可以尝试使用 `scan` 命令手动查看工具检测到的路径是否正确。

---

## 📩 联系方式

加微信请备注来意。**注意：1+1 这种基础问题（如：如何打开命令行、如何安装 Go 等）概不回复，请自行搜索。**

<img src="https://i.imgur.com/9PxS5IK.jpeg" width="300" />

---
ß
## ☕ 请我喝咖啡

如果这个工具帮助了你，欢迎请我喝杯咖啡 ☕，这将是对我持续更新的最大动力！

### 💝 赞助记录

| 日期 | 方式 | 备注 | 金额 |
|------|------|------|------|
| 2026/04/17 | WeChat | UR的出不克 | 50 CNY |

感谢每一位支持者！🙏

---

## 🌟 Star History

如果这个项目对你有帮助，请给一个 ⭐ Star！

---

<div align="center">

**Made with ❤️ by [25smoking](https://github.com/25smoking)**

</div>
