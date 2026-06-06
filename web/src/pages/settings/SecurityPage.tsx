import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link2, Unlink } from 'lucide-react'
import { toast } from 'sonner'
import { api, oidcStartUrl } from '../../api/client'
import { EmptyState } from '../../components/common/empty-state'
import { ErrorState } from '../../components/common/error-state'
import { MotionItem, MotionList } from '../../components/common/motion'
import { PageHeader } from '../../components/common/page-header'
import { StatusBadge } from '../../components/common/status-badge'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'

export function SecurityPage() {
  const queryClient = useQueryClient()
  const providers = useQuery({ queryKey: ['auth-providers'], queryFn: () => api.listAuthProviders(false) })
  const identities = useQuery({ queryKey: ['external-identities'], queryFn: api.listMyExternalIdentities })
  const unbind = useMutation({
    mutationFn: api.unbindMyExternalIdentity,
    onSuccess: () => {
      toast.success('第三方登录已解绑')
      queryClient.invalidateQueries({ queryKey: ['external-identities'] })
    },
    onError: error => toast.error(error.message),
  })

  return (
    <div className="grid gap-6">
      <PageHeader
        description="管理当前账号绑定的 OIDC 第三方登录。"
        title="账号安全"
      />

      <div className="grid gap-4 lg:grid-cols-[1fr_360px]">
        <Card>
          <h2 className="mb-4 text-base font-semibold">已绑定身份</h2>
          {identities.isError && <ErrorState title="身份加载失败" description="请刷新页面后重试。" />}
          {identities.data?.length === 0 && <EmptyState title="还没有绑定第三方登录" description="从右侧选择一个已启用身份源进行绑定。" />}
          <MotionList className="grid gap-3">
            {(identities.data ?? []).map(identity => (
              <MotionItem key={identity.id}>
                <div className="flex items-center justify-between gap-4 rounded-md border border-border p-3 transition duration-150 hover:border-primary hover:shadow-sm">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="font-medium">{identity.providerName}</p>
                      <StatusBadge>{identity.emailVerified ? 'verified' : 'unverified'}</StatusBadge>
                    </div>
                    <p className="truncate text-sm text-muted-foreground">{identity.email || identity.username || identity.subject}</p>
                  </div>
                  <Button
                    aria-label="解绑第三方登录"
                    disabled={unbind.isPending}
                    variant="ghost"
                    onClick={() => unbind.mutate(identity.id)}
                  >
                    <Unlink size={16} />
                  </Button>
                </div>
              </MotionItem>
            ))}
          </MotionList>
        </Card>

        <Card>
          <h2 className="mb-4 text-base font-semibold">绑定新身份源</h2>
          {providers.isError && <ErrorState title="身份源加载失败" description="请稍后重试。" />}
          <div className="grid gap-2">
            {(providers.data ?? []).map(provider => (
              <Button
                key={provider.id}
                type="button"
                variant="secondary"
                onClick={() => {
                  window.location.href = oidcStartUrl(provider.id, 'bind', '/settings/security')
                }}
              >
                <Link2 size={16} />
                绑定
                {' '}
                {provider.name}
              </Button>
            ))}
          </div>
        </Card>
      </div>
    </div>
  )
}
