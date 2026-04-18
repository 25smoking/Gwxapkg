# Release v2.6.0 - 回包能力增强

## 主要更新

### 新功能

#### 1. 支持生成微信客户端可识别的加密回包
- `repack` 新增 `-id=<AppID>`，可输出加密后的 `V1MMWX` 包
- 新增 `-raw`，可按需输出未加密测试包
- 回包结果可再次被工具解开验证

#### 2. 支持按原始多包结构精确回包
- 解包时会记录原始包清单 `manifest`
- 回包时优先按 `manifest` 恢复多包结构
- 支持输出 `__APP__.wxapkg`、子包等多个原始包，而不是强制合成单包

#### 3. 新增 `workspace` 工作区模式
- 解包并还原源码时，可额外保留隐藏的原始运行时文件
- 默认存放在 `.gwxapkg/raw/<包名>/`
- 后续可直接对同一个输出目录执行 `repack`
- 适合“解包 -> 修改 -> 回包”的连续工作流

### 修复问题

#### 1. 修复 macOS 新版微信缓存目录扫描失败
- 兼容新版路径：
  `~/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/users/*/applet/packages`
- 解决“找不到正确目录信息”的问题

#### 2. 修复重打包后客户端无法打开的问题
- 旧版 `repack` 输出的是明文 `wxapkg`
- 现在提供正确的微信加密封装流程
- 更接近客户端真实使用场景

#### 3. 修复输出目录 `~` 不展开的问题
- `-out=~/xxx` 现在会自动展开到用户主目录
- 对解包和 `repack` 都生效

#### 4. 修复包信息并发访问风险
- 为 `WxapkgManager` 增加并发保护
- 避免多文件并行处理时的 map 并发读写问题

### 使用体验优化

- `repack` 支持从隐藏工作区自动读取原始文件
- 普通单包打包流程会自动排除 `.gwxapkg` 工作区内容
- 还原流程会跳过隐藏工作区，避免误删原始回包素材

## 推荐用法

```bash
# 1. 解包并保留可回包工作区
./gwxapkg -id=<AppID> -in=<原始目录> -out=<工作目录> -restore=true -workspace=true

# 2. 直接从工作目录回包
./gwxapkg repack -in=<工作目录> -out=<输出目录> -id=<AppID>
```

## 说明

- `restore=true` 输出的源码目录本身并不是官方编译产物
- 当前可回包能力依赖隐藏保留的原始运行时文件工作区
- 如需从同一目录回包，请启用 `-workspace=true`

## 下载

### macOS (Apple Silicon)
- `gwxapkg-darwin-arm64`

### Windows (64-bit)
- `gwxapkg-windows-amd64.exe`
