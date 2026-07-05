import type { NotificationChannel, NotificationChannelPayload, NotificationDelivery, NotificationPreset, NotificationRule, NotificationRulePayload, NotificationTemplate, NotificationTemplatePayload, PaginatedResponse, PaginationParams } from '../types'
import { paginationQuery, request } from '../core'

export const notificationsApi = {
  listNotificationPresets: () => request<NotificationPreset[]>('/notifications/presets'),
  createNotificationChannelFromPreset: (presetId: string, payload: { name: string, secrets: Record<string, string>, enabled: boolean }) =>
    request<{ channel: NotificationChannel, template: NotificationTemplate }>(`/notifications/presets/${encodeURIComponent(presetId)}/channels`, { method: 'POST', body: JSON.stringify(payload) }),
  listNotificationChannels: (params: PaginationParams) =>
    request<PaginatedResponse<NotificationChannel>>(`/notifications/channels?${paginationQuery(params)}`),
  createNotificationChannel: (payload: NotificationChannelPayload) =>
    request<NotificationChannel>('/notifications/channels', { method: 'POST', body: JSON.stringify(payload) }),
  updateNotificationChannel: (channelId: string, payload: NotificationChannelPayload) =>
    request<NotificationChannel>(`/notifications/channels/${channelId}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteNotificationChannel: (channelId: string) =>
    request<void>(`/notifications/channels/${channelId}`, { method: 'DELETE' }),
  testNotificationChannel: (channelId: string) =>
    request<{ status: string }>(`/notifications/channels/${channelId}/test`, { method: 'POST' }),
  listNotificationTemplates: (params: PaginationParams) =>
    request<PaginatedResponse<NotificationTemplate>>(`/notifications/templates?${paginationQuery(params)}`),
  createNotificationTemplate: (payload: NotificationTemplatePayload) =>
    request<NotificationTemplate>('/notifications/templates', { method: 'POST', body: JSON.stringify(payload) }),
  updateNotificationTemplate: (templateId: string, payload: NotificationTemplatePayload) =>
    request<NotificationTemplate>(`/notifications/templates/${templateId}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteNotificationTemplate: (templateId: string) =>
    request<void>(`/notifications/templates/${templateId}`, { method: 'DELETE' }),
  listNotificationRules: (params: PaginationParams) =>
    request<PaginatedResponse<NotificationRule>>(`/notifications/rules?${paginationQuery(params)}`),
  createNotificationRule: (payload: NotificationRulePayload) =>
    request<NotificationRule>('/notifications/rules', { method: 'POST', body: JSON.stringify(payload) }),
  updateNotificationRule: (ruleId: string, payload: NotificationRulePayload) =>
    request<NotificationRule>(`/notifications/rules/${ruleId}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteNotificationRule: (ruleId: string) =>
    request<void>(`/notifications/rules/${ruleId}`, { method: 'DELETE' }),
  listNotificationDeliveries: (params: PaginationParams) =>
    request<PaginatedResponse<NotificationDelivery>>(`/notifications/deliveries?${paginationQuery(params)}`),
}
