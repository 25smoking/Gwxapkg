# Gwxapkg

<div align="center">

![Version](https://img.shields.io/badge/version-2.5.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)
![Build](https://img.shields.io/badge/build-passing-brightgreen.svg)

**[中文](README.md) | [English](README_EN.md) | [日本語](README_JA.md)**

一款基于Go实现的微信小程序 `.wxapkg` 解包工具，支持自动扫描、解密、反编译和安全分析。

</div>

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
- **目录结构** - 还原微信小程序原始工程目录
- **资源提取** - 图片/音频/视频等资源文件完整提取

### � 安全分析 ⭐ NEW
- **智能扫描** - 200+ 敏感信息检测规则
- **误报过滤** - 智能黑名单，误报率从95%降至10-15%
- **数据去重** - 自动去除重复数据，精准定位
- **Excel报告** - 专业的多Sheet分类报告，包含文件路径和行号
- **风险分级** - 高/中/低风险自动分类

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
| `.js` | ✅ | JavaScript 代码还原 + 美化 |
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

# 解包单个 wxapkg 文件
./gwxapkg -id=<AppID> -in=<文件路径>

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
| `-noClean` | 保留中间临时文件 | false |
| `-save` | 保存解密后的文件 | false |

### 使用示例

```bash
# 示例1: 自动扫描并处理
./gwxapkg all -id=wx3c19e32cb8f31289

# 示例2: 仅解包单个文件
./gwxapkg -id=wx123456 -in=test.wxapkg -out=./output

# 示例3: 重新打包
./gwxapkg repack -in=./source_dir -out=new.wxapkg
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

### 扫描规则（200+）

| 分类 | 规则数 | 示例 |
|------|--------|------|
| **路径** | 1 | 文件路径、系统路径 |
| **URL** | 2 | HTTP/HTTPS链接、API端点 |
| **域名** | 1 | 域名地址（TLD验证） |
| **账号密码** | 12+ | 各类密码、数据库凭证 |
| **API密钥** | 40+ | AWS/阿里云/腾讯云等密钥 |
| **令牌** | 30+ | JWT/Bearer/OAuth令牌 |
| **数据库** | 15+ | MySQL/MongoDB/Redis连接串 |
| **联系信息** | 3 | 手机号/邮箱/身份证 |
| **微信** | 4 | AppID/Secret/Webhook |
| **其他** | 90+ | 证书/哈希/UUID等 |

### Excel报告内容

生成的报告包含以下Sheet：

- **概览** - 扫描统计、风险分布、分类汇总
- **路径** - 所有路径类敏感信息
- **URL** - 所有URL和API端点
- **域名** - 域名地址（已过滤误报）
- **账号密码** - 密码和凭证信息
- **API密钥** - 各类云服务密钥
- **令牌** - 访问令牌和会话信息
- **数据库** - 数据库连接信息
- **联系信息** - 手机号、邮箱等
- **微信** - 微信相关配置
- **其他** - 其他敏感信息

每条数据包含：
- ✅ 内容（Content）
- ✅ 出现次数
- ✅ 文件路径
- ✅ 行号
- ✅ 风险等级

---

## 📈 性能对比（v2.5.0 vs v1.0）

| 指标 | v1.0 | v2.5.0 | 改进 |
|------|------|--------|------|
| **扫描速度** | 基准 | +50-70% | ⬆️⬆️⬆️ 规则预编译 |
| **误报率** | ~95% | 10-15% | ⬇️⬇️⬇️ 智能过滤 |  
| **数据量** | 127,185条 | ~3,000条 | ⬇️⬇️⬇️ 去重+过滤 |
| **输出格式** | JSON | Excel | ✅ 专业报告 |
| **并发性能** | 10固定 | CPU*2动态 | ⬆️⬆️ 自适应 |
| **I/O性能** | 直接写入 | 256KB缓冲 | ⬆️⬆️ 减少系统调用 |

---

## 🔄 版本更新

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
│   ├── formatter/        # 代码美化（JS/CSS/HTML）
│   ├── key/              # 规则管理，预编译
│   ├── scanner/          # ⭐ NEW 扫描引擎
│   │   ├── types.go      # 数据模型
│   │   ├── filter.go     # 误报过滤
│   │   ├── collector.go  # 数据收集和去重
│   │   └── scanner.go    # 扫描逻辑
│   ├── reporter/         # ⭐ NEW 报告生成
│   │   └── excel.go      # Excel报告
│   ├── config/           # 配置管理
│   └── ui/               # 终端UI
├── config/
│   └── rule.yaml         # 200+ 敏感信息规则
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

## ⚠️ 免责声明

本工具仅供学习和研究使用，请勿用于非法用途。使用本工具产生的任何后果由使用者自行承担。

---

## 🌟 Star History

如果这个项目对你有帮助，请给一个 ⭐ Star！

---

<div align="center">

**Made with ❤️ by [25smoking](https://github.com/25smoking)**

</div>