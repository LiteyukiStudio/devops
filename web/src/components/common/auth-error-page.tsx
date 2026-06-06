import { Link } from 'react-router-dom'
import { Button } from '../ui/button'
import { Card } from '../ui/card'

export function AuthErrorPage({ title, description }: { title: string, description: string }) {
  return (
    <div className="grid min-h-screen place-items-center bg-background px-4 text-foreground">
      <Card className="w-full max-w-md">
        <h1 className="text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        <Button className="mt-5">
          <Link to="/login">返回登录</Link>
        </Button>
      </Card>
    </div>
  )
}
