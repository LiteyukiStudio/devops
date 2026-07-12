import { describe, expect, it, vi } from 'vitest'
import { openDeploymentTargetDataExport } from './deployment-target-data-export'

function exportWindow() {
  return {
    close: vi.fn(),
    location: { replace: vi.fn() },
    opener: window,
  }
}

describe('openDeploymentTargetDataExport', () => {
  it('opens a blank tab before authorizing and navigates it after authorization', async () => {
    let resolveAuthorization!: () => void
    const authorization = new Promise<{ ticket: string, expiresAt: string }>((resolve) => {
      resolveAuthorization = () => resolve({ ticket: 'ticket with spaces', expiresAt: '2026-07-12T00:01:00Z' })
    })
    const authorize = vi.fn(() => authorization)
    const targetWindow = exportWindow()
    const openWindow = vi.fn(() => targetWindow)

    const exporting = openDeploymentTargetDataExport('project one', 'app/two', 'target three', { authorize, openWindow })

    expect(openWindow).toHaveBeenCalledOnce()
    expect(targetWindow.opener).toBeNull()
    expect(authorize).toHaveBeenCalledWith('project one', 'app/two', 'target three')
    expect(targetWindow.location.replace).not.toHaveBeenCalled()

    resolveAuthorization()
    await exporting

    expect(targetWindow.location.replace).toHaveBeenCalledWith('/api/v1/projects/project%20one/applications/app%2Ftwo/deployment-targets/target%20three/data-export?ticket=ticket+with+spaces')
    expect(targetWindow.close).not.toHaveBeenCalled()
  })

  it('closes the blank tab when authorization is cancelled or fails', async () => {
    const error = new Error('mfa_challenge_cancelled')
    const authorize = vi.fn(async () => Promise.reject(error))
    const targetWindow = exportWindow()

    await expect(openDeploymentTargetDataExport('project', 'app', 'target', {
      authorize,
      openWindow: () => targetWindow,
    })).rejects.toBe(error)

    expect(targetWindow.close).toHaveBeenCalledOnce()
    expect(targetWindow.location.replace).not.toHaveBeenCalled()
  })

  it('does not authorize when the browser blocks the blank tab', async () => {
    const authorize = vi.fn(async () => ({ ticket: 'unused', expiresAt: '2026-07-12T00:01:00Z' }))

    await expect(openDeploymentTargetDataExport('project', 'app', 'target', {
      authorize,
      openWindow: () => null,
    })).rejects.toThrow('data_export_window_blocked')

    expect(authorize).not.toHaveBeenCalled()
  })
})
