# 品牌与 Logo

## Logo 文件

- 主 Logo：`web/public/liteyuki-logo.svg`
- Mascot：`web/public/brand/mascot-liteyuki-devops.png`

## 设计定位

Liteyuki DevOps 是面向个人开发者和小团队的应用交付平台。Logo 采用方形图标布局，优先适配站点 Logo、favicon、应用图标和控制台侧边栏等小尺寸场景。

## 图形语义

- 六边形容器：对应镜像、Kubernetes 资源和平台统一托管的运行单元。
- 六辐船舵底纹：参考 Kubernetes 船舵语义，但由七辐改为六辐，使其贴合自有六边形容器。
- 曲线路径与三个节点：对应从代码仓库到构建、部署、网关域名的 DevOps 交付闭环。
- 字母 L：保留 Liteyuki 的品牌识别，去除横版字标后仍可在小尺寸下识别。
- 右上分层面板：抽象表达镜像层、构建产物和运行配置。

## 色彩

- 青绿色 `#22C7A9`：表示轻量、可达和流程通过。
- 蓝色 `#3478F6`：表示平台、基础设施和可靠交付。
- 靛蓝 `#5B5FEF`：表示技术控制台和云原生环境。
- 暖黄色 `#F6B73C`：表示触发点、发布入口和用户行动。

控制台主色：

- MVP 控制台主色采用 Kubernetes 风格蓝 `#326CE5`，与 Logo 中的蓝色节点和云原生语义保持一致。
- 暗色主题主色使用更亮的 `#60A5FA`，保证深色背景上的按钮、激活态和焦点环可读。
- 青绿色保留为 Logo 辅助色和成功/流程通过语义，不作为控制台默认主色。

## 使用建议

- 站点配置中的 `site.logoUrl` 可使用 `/liteyuki-logo.svg`。
- 站点配置中的 `site.faviconUrl` 也默认使用 `/liteyuki-logo.svg`。
- 该版本已经是方形图标，可直接作为 favicon 或移动端图标的基础版本。
- 项目只维护这一份 SVG 源文件，README、favicon 和前端默认配置都应引用它，避免多份资产后续不同步。
- Mascot 适合用于登录页、README 展示和空状态插画，不建议作为小尺寸图标使用。
