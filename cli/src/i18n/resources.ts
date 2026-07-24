export const resources = {
  'en-US': {
    translation: {
      common: {
        empty: 'No items found.',
        error: 'Command failed.',
        warning: 'Warning',
      },
      cli: {
        description: 'Luna DevOps command-line client for people and agents',
      },
      confirm: {
        execute: 'Run this command?',
      },
      errors: {
        invalid_arguments: 'Input validation failed.',
        unauthenticated: 'Authentication is required.',
        forbidden: 'You do not have permission to perform this operation.',
        not_found: 'The requested resource was not found.',
        conflict: 'The resource changed or is not in the required state.',
        retry_later: 'The operation cannot be completed yet. Try again later.',
        service_failure: 'The service is temporarily unavailable.',
        confirmation_required: 'This operation requires confirmation.',
        operation_cancelled: 'Operation cancelled.',
        server_plan_required: 'This high-risk operation requires a server-issued execution plan.',
      },
      table: {
        name: 'Name',
        status: 'Status',
        type: 'Type',
        createdAt: 'Created',
        updatedAt: 'Updated',
      },
    },
  },
  'zh-CN': {
    translation: {
      common: {
        empty: '没有找到数据。',
        error: '命令执行失败。',
        warning: '警告',
      },
      cli: {
        description: '面向用户和智能体的 Luna DevOps 命令行客户端',
      },
      confirm: {
        execute: '确认执行此命令吗？',
      },
      errors: {
        invalid_arguments: '输入参数校验失败。',
        unauthenticated: '需要先完成身份验证。',
        forbidden: '当前账号没有执行此操作的权限。',
        not_found: '未找到请求的资源。',
        conflict: '资源已发生变化或当前状态不允许此操作。',
        retry_later: '当前暂时无法完成操作，请稍后重试。',
        service_failure: '服务暂时不可用。',
        confirmation_required: '此操作需要先确认。',
        operation_cancelled: '操作已取消。',
        server_plan_required: '此高风险操作需要服务端签发执行计划。',
      },
      table: {
        name: '名称',
        status: '状态',
        type: '类型',
        createdAt: '创建时间',
        updatedAt: '更新时间',
      },
    },
  },
} as const

export type SupportedLocale = keyof typeof resources
export const SUPPORTED_LOCALES = Object.freeze(Object.keys(resources) as SupportedLocale[])
