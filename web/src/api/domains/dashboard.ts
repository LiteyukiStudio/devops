import type { DashboardOverview } from '../types'
import { request } from '../core'

export const dashboardApi = {
  getDashboard: () => request<DashboardOverview>('/dashboard'),
}
