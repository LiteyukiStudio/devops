const appTemplatesPage = {
  description: '从预设模板一键安装 Redis、PostgreSQL、MySQL 等常用基础应用。',
  heroTitle: '应用市场',
  heroDescription: '选择一个内置模板，填写少量参数后安装到项目空间；平台会创建应用、部署配置，并按需立即发布。',
  searchPlaceholder: '搜索模板、镜像或分类',
  loading: '正在加载应用模板...',
  emptyTitle: '没有找到模板',
  emptyDescription: '换个关键词试试，或等待管理员添加更多模板。',
  install: '安装',
  installing: '安装中...',
  installStarted: '应用模板已开始安装',
  installDialogTitle: '安装 {{name}}',
  installDialogDescription: '模板会在目标项目空间中创建一个应用和部署配置；密钥类参数会安全存储，不会明文回显。',
  runtimeCluster: '运行集群',
  defaultCluster: '使用默认集群',
  applicationName: '应用名称',
  applicationSlug: '应用标识',
  deploymentName: '部署配置',
  stage: '阶段',
  replicas: '副本数',
  cpu: 'CPU',
  memory: '内存',
  dataCapacity: '数据容量',
  templateParameters: '模板参数',
  templateParametersDescription: '留空的自动生成密钥会由后端生成并写入密钥存储。',
  autoGeneratePlaceholder: '留空自动生成',
  installNow: '安装后立即部署',
  installNowDescription: '关闭后只创建应用和部署配置，之后可在应用部署页手动发布。',
  image: '镜像',
  port: '端口',
  resources: '资源',
  categories: {
    database: '数据库',
    middleware: '中间件',
  },
  stageOptions: {
    prod: '生产',
    staging: '预发',
    test: '测试',
    dev: '开发',
  },
  valueLabels: {
    username: '用户名',
    database: '数据库名',
    password: '密码',
    rootPassword: 'Root 密码',
  },
  templates: {
    redis: {
      description: '用于缓存、队列和轻量协调的内存数据存储。',
    },
    postgresql: {
      description: '适合应用数据、元数据和事务型场景的关系型数据库。',
    },
    mysql: {
      description: '经典关系型数据库，使用单容器快速安装。',
    },
    rabbitmq: {
      description: '用于异步任务、事件和轻量消息队列的消息中间件。',
    },
  },
}

export default appTemplatesPage
