import { Link } from 'react-router-dom'
import { Card } from '../ui/card'

export function ForbiddenPage() {
  return (
    <div className="grid min-h-screen place-items-center bg-background px-4 text-foreground">
      <Card className="w-full max-w-md">
        <h1 className="text-xl font-semibold">没有访问权限</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          你当前不是该项目成员，或没有执行此操作的权限。请返回项目列表，或联系项目管理员授予权限。
        </p>
        <Link className="mt-5 inline-flex h-9 items-center justify-center rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground transition hover:bg-primary/90" to="/projects">
          返回项目
        </Link>
      </Card>
    </div>
  )
}
