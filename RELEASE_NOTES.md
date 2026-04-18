# Release v2.7.0 - 功能增强

## 主要更新

### 新功能

#### 1. 支持 Windows PC 端提取应用真实名称
- 针对最新版微信缓存隔离沙盒（`radium/users/*/applet/packages`）带来的路径解析难点
- 扫描阶段动态内存拦截 `__APP__.wxapkg`，直读 `app-config.json` 获取 `navigationBarTitleText`
- 纯 Golang 实现极速处理，执行 `scan` 命令时终端列表可直观展示多套中文应用名称

#### 2. 全新交互式 HTML 安全审查报告
- 新增 HTML 报告生成格式，相比原有单一输出增加更多可视化能力
- 内置敏感词大类分布饼图、可实时检索过滤漏洞、根据规则配置分类多选项卡
- 单文件架构，无需额外加载资源或本地启动依赖服务

#### 3. 新增 `scan-only` 独立安全审查模式
- 专门用于针对已解包的纯净源码目录发起深层信息扫描
- 与解包流程解耦，方便用户对自行获取的其他项目源代码作快速测试查违点
- 同样支持自动化输出完整的 HTML 图表化安全检测报告

#### 4. `all` 批量处理指令支持指定 `-id`
- 新增 `-id="wx1,wx2"` 参数通过逗号隔开进行多目标筛选，告别暴力枚举
- 新增 `-id-file=ids.txt` 指定外部文件批量逐行读取并联动流水线
- 支持大型集群环境的 CI 部署与渗透任务分发


### 修复问题

#### 1. 修复老版本无法连贯读取部分 Windows PC 下程序包状态
- 兼容最新多层沙盒嵌套路径提取
- 避开微信官方新引入 SQLite 固化所产生的不连贯读取断档失败情况

### 使用体验优化

- `scan` 选择终端输出经过优化容错处理，展示形态更加美观直白
- `all` 内部调度协程增强防护，避免多 AppID 处理状态时终端信息互相重叠

## 推荐用法

```bash
# 1. 扫描所有缓存的小程序列表（终端即能清楚看到对应的真实中文名）
./gwxapkg scan

# 2. 定点批量解包，并在目标目录下生成详尽的安全 HTML 分析报告
./gwxapkg all -id="wx123,wx456" -sensitive=true

# 3. 剥离式独立审查漏洞复审历史解包文件
./gwxapkg scan-only -in=./history_source_code/ -out=./reports/
```

## 说明

- Windows 小游戏及第三方插件组件依赖代码中，由于无标准前端开发入口规范（如缺乏明确标准 app-config），终端获取展示时可能为空属于正常现象。
- 所有名称提取及审查过程只热解析暂且所需的片区数据流，不会额外在本地持久性留存解压遗留产生的垃圾文件。

## 下载

### macOS (Apple Silicon & Intel)
- `gwxapkg-darwin-arm64`
- `gwxapkg-darwin-amd64`

### Windows (64-bit)
- `gwxapkg-windows-amd64.exe`

### Linux (64-bit)
- `gwxapkg-linux-amd64`
