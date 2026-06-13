export type StatusTone = 'danger' | 'info' | 'neutral' | 'success' | 'warning'

export function statusToneFor(value: string): StatusTone {
  switch (value.trim().toLowerCase()) {
    case 'active':
    case 'connected':
    case 'created':
    case 'enabled':
    case 'healthy':
    case 'issued':
    case 'passed':
    case 'ready':
    case 'succeeded':
    case 'success':
    case 'verified':
      return 'success'
    case 'failed':
    case 'crash-loop-back-off':
    case 'create-container-config-error':
    case 'create-container-error':
    case 'delete_failed':
    case 'err-image-pull':
    case 'image-pull-back-off':
    case 'lost':
    case 'missing-credential':
    case 'revoked':
    case 'timeout':
    case 'unhealthy':
      return 'danger'
    case 'expired':
    case 'checking':
    case 'container-creating':
    case 'pending':
    case 'progressing':
    case 'queued':
    case 'deleting':
    case 'running':
    case 'scanning':
    case 'not-ready':
      return 'warning'
    case 'createdstatus':
      return 'success'
    case 'disabled':
    case 'canceled':
    case 'not-configured':
    case 'not-found':
    case 'unknown':
      return 'neutral'
    default:
      return 'info'
  }
}
