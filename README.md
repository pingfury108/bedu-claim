# 百度教育任务自动认领系统 🎯

一个基于 Wails 开发的跨平台桌面应用，用于自动化百度教育平台任务的认领过程，支持 Windows 和 macOS 系统。

## 🌟 项目简介

本系统专为百度教育平台的任务管理而设计，支持自动认领审核任务和生产任务，具备智能筛选、关键词过滤、时间段控制等功能，帮助用户高效管理教育内容审核工作。

## ✨ 核心功能

### 🔍 智能任务筛选
- **任务类型支持**: 审核任务 (audittask) 和生产任务 (producetask)
- **多维度筛选**: 学段、学科、任务类型
- **实时标签数据**: 从百度教育平台获取最新筛选条件

### ⚙️ 高级过滤功能
- **关键词过滤**: 支持包含关键词和排除关键词
- **时间范围**: 精确控制任务发布时间范围
- **认领上限**: 自定义单次认领任务数量限制
- **轮询间隔**: 灵活设置检查频率

### 👤 用户认证系统
- **Cookie验证**: 基于百度教育平台Cookie的身份认证
- **用户信息**: 实时显示当前登录用户信息
- **权限管理**: 通过LLM测试端点验证用户权限

### 📊 实时监控
- **状态跟踪**: 实时显示认领进度和状态
- **成功统计**: 记录成功认领的任务数量
- **错误日志**: 详细记录操作过程中的错误信息

## 🚀 快速开始

### 📋 环境要求

- **Go**: 1.23 或更高版本
- **Node.js**: 18 或更高版本
- **操作系统**: Windows 10/11, macOS 10.15+

### 🔧 安装步骤

#### 1. 克隆项目
```bash
git clone https://github.com/your-username/bedu-claim.git
cd bedu-claim
```

#### 2. 安装依赖
```bash
# 安装Go依赖
go mod tidy

# 安装前端依赖
cd frontend
npm install
```

#### 3. 开发模式运行
```bash
# 启动开发服务器
wails dev
```

#### 4. 构建生产版本
```bash
# 构建所有支持的平台
wails build

# 构建特定平台
wails build --platform windows/amd64
wails build --platform darwin/amd64
wails build --platform darwin/arm64
```

### 📦 下载预构建版本

访问 [Releases 页面](https://github.com/your-username/bedu-claim/releases) 下载适用于您平台的预构建版本。

## 🎯 使用指南

### 1. 获取Cookie
1. 登录 [百度教育平台](https://easylearn.baidu.com)
2. 打开浏览器开发者工具 (F12)
3. 切换到 Application/Storage 标签页
4. 找到 Cookie 并复制相关值

### 2. 配置系统
1. 启动应用程序
2. 在"百度教育 Cookie"输入框中粘贴Cookie
3. 系统将自动验证Cookie有效性并显示用户信息
4. 选择任务类型（审核/生产）
5. 配置筛选条件（学段、学科、类型）
6. 设置认领参数

### 3. 启动自动认领
1. 设置认领上限（建议10-50个）
2. 配置轮询间隔（建议1-5秒）
3. 可选：设置关键词过滤和时间范围
4. 点击"启动自动认领"按钮

### 4. 监控与管理
- 实时查看认领状态和成功数量
- 查看错误日志进行故障排查
- 随时停止自动认领过程

## ⚙️ 配置参数

| 参数 | 说明 | 示例 |
|------|------|------|
| Cookie | 百度教育平台身份凭证 | `BDUSS=xxx; STOKEN=xxx` |
| 任务类型 | 审核任务或生产任务 | `audittask` / `producetask` |
| 认领上限 | 单次认领最大数量 | `10` |
| 轮询间隔 | 检查新任务频率 | `1.5` 秒 |
| 关键词过滤 | 包含/排除特定关键词 | `数学,英语` |
| 时间范围 | 任务发布时间过滤 | `2024-01-01 00:00 - 2024-01-31 23:59` |

## 🛠️ 技术架构

### 技术栈
- **后端**: Go 1.23 + Wails v2
- **前端**: React 18 + TypeScript + Tailwind CSS + DaisyUI
- **构建工具**: Vite
- **跨平台**: Windows, macOS

### 项目结构
```
bedu-claim/
├── app.go              # 主应用逻辑
├── bedu_api.go         # 百度教育API接口
├── main.go             # 应用入口
├── frontend/           # 前端代码
│   ├── src/
│   │   ├── App.tsx     # 主组件
│   │   ├── ClueClaimingComponent.tsx  # 任务认领组件
│   │   └── ...
│   ├── package.json    # 前端依赖
│   └── vite.config.ts  # Vite配置
├── .github/workflows/  # GitHub Actions工作流
└── wails.json          # Wails配置
```

## 🔒 安全特性

- **本地存储**: Cookie仅在本地存储，不上传服务器
- **权限验证**: 多重权限验证机制
- **错误处理**: 完善的错误处理和用户提示
- **日志记录**: 详细的操作日志便于审计

## 🐛 故障排除

### 常见问题

#### 1. Cookie验证失败
**症状**: 显示"无权使用该软件"
**解决**: 
- 确认Cookie格式正确
- 检查Cookie是否过期
- 验证网络连接是否正常

#### 2. 无法获取任务列表
**症状**: 标签数据加载失败
**解决**:
- 检查网络连接
- 确认Cookie有效性
- 查看是否为工作时间

#### 3. 认领失败
**症状**: 认领状态显示错误
**解决**:
- 检查认领参数设置
- 确认任务类型选择正确
- 查看是否有足够的任务数量

### 调试模式
```bash
# 启用调试日志
wails dev -loglevel debug
```

## 🔄 GitHub Actions

项目配置了自动构建工作流，支持以下平台：
- Windows AMD64
- Windows ARM64  
- macOS AMD64
- macOS ARM64

每次推送到 main 分支或创建 release 时，会自动构建对应平台的可执行文件。

## 🔄 更新日志

### v1.0.0 (2024-08-04)
- ✨ 初始版本发布
- 🎯 支持审核任务自动认领
- 🔍 基础关键词过滤功能
- 👤 用户身份验证系统
- 📊 实时状态监控
- 🖥️ 支持 Windows 和 macOS 平台

## 🤝 贡献指南

欢迎提交Issue和Pull Request来改进这个项目！

### 开发环境搭建
1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 代码规范
- 使用 `gofmt` 格式化Go代码
- 使用 `eslint` 和 `prettier` 格式化前端代码
- 遵循语义化版本控制

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 👥 联系方式

- **项目维护者**: pingfury
- **邮箱**: pingfury@outlook.com
- **问题反馈**: [GitHub Issues](https://github.com/your-username/bedu-claim/issues)

## 🙏 致谢

- [Wails](https://wails.io/) - 跨平台应用开发框架
- [百度教育平台](https://easylearn.baidu.com) - 提供API接口支持
- [DaisyUI](https://daisyui.com/) - 优雅的UI组件库

---

<div align="center">
  <p><strong>⭐ 如果这个项目对你有帮助，请给个Star！</strong></p>
  <p><sub>Built with ❤️ by the bedu-claim team</sub></p>
</div>
