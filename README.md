# Gwxapkg

一款基于GO实现的微信小程序 `.wxapkg` 解包工具，支持自动扫描、解密、反编译，小程序安全测试。

## ✨ 特性

- 🔍 **自动扫描** - 自动检测 macOS / Windows 微信小程序缓存目录
- 🔓 **自动解密** - 支持加密的 wxapkg 文件解密
- 📦 **一键解包** - 自动查找并处理指定 AppID 的所有文件
- 🎨 **美化输出** - 彩色终端输出，清晰的进度显示
- 🔄 **重新打包** - 支持将修改后的文件重新打包

## 🛠️ 功能说明

### 解包能力

| 文件类型 | 支持情况 | 说明 |
|----------|----------|------|
| `.wxml` | ✅ | 页面结构还原 |
| `.wxss` | ✅ | 样式文件还原 |
| `.js` | ✅ | JavaScript 代码还原 + 美化 |
| `.json` | ✅ | 配置文件提取 |
| `.wxs` | ✅ | WXS 脚本还原 |
| 图片/音频/视频 | ✅ | 资源文件提取 |

### 核心功能

- **wxapkg 解包** - 解析 wxapkg 二进制格式，提取所有文件
- **代码美化** - 自动格式化 JS/CSS 代码，提高可读性
- **目录还原** - 还原微信小程序的原始工程目录结构
- **加密解密** - 支持 PC 端加密的 wxapkg 文件
- **分包处理** - 正确处理主包和分包的依赖关系
- **敏感信息扫描** - 提取 AppID、密钥等敏感数据

### 软件特点

- 🚀 **高性能** - Go 语言编写，并发处理多个文件
- 💻 **跨平台** - 支持 macOS、Windows、Linux
- 🎯 **零依赖** - 单文件可执行，无需安装运行时
- 🔧 **易于使用** - 简洁的命令行界面，开箱即用
- 📱 **智能识别** - 自动识别文件类型和加密方式

## 📥 安装

### 从源码编译

```bash
git clone https://github.com/25smoking/Gwxapkg.git
cd Gwxapkg
go build -o Gwxapkg .
```

### 下载预编译版本

前往 [Releases](https://github.com/25smoking/Gwxapkg/releases) 下载。

## 🚀 使用方法

### 扫描本地小程序

```bash
./Gwxapkg scan
```

输出示例：
```
✓ 找到 66 个小程序
─────────────────────────────────────────────────────

   1. wx3c19e32cdsad21289
     版本: 66 │ 文件: 7 │ 更新: 2025-12-04 15:07
     路径: ~/Library/.../packages/wx3c19e32cdsad21289/66
```

### 自动处理指定小程序

```bash
./Gwxapkg all -id=wx3c19e32cdsad21289 -out=./output
```

### 手动指定文件

```bash
./Gwxapkg -id=<AppID> -in=<文件路径> -out=<输出目录>
```

### 重新打包

```bash
./Gwxapkg repack -in=<目录> [-watch]
```

## ⚙️ 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-out` | 输出目录 | 自动 |
| `-restore` | 还原目录结构 | `true` |
| `-pretty` | 美化代码输出 | `true` |
| `-noClean` | 保留中间文件 | `false` |
| `-save` | 保存解密文件 | `false` |
| `-sensitive` | 获取敏感数据 | `true` |

## 📁 支持的路径

### macOS
- `~/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages`
- `~/Library/Application Support/WeChat/Applet/packages`

### Windows
- `%APPDATA%/Tencent/xwechat/radium/Applet/packages`
- `Documents/WeChat Files/Applet`

## 🙏 致谢

本项目基于 [KillWxapkg](https://github.com/Ackites/KillWxapkg) 二次开发，感谢原作者 [@Ackites](https://github.com/Ackites) 的杰出工作。

## 📜 许可证

MIT License