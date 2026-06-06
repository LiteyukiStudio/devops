import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Save } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { api } from '../../api/client'
import { ErrorState } from '../../components/common/error-state'
import { PageHeader } from '../../components/common/page-header'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Field, Input } from '../../components/ui/input'

export function SiteSettingsPage() {
  const queryClient = useQueryClient()
  const definitions = useQuery({ queryKey: ['config-definitions'], queryFn: api.listConfigDefinitions })
  const keys = useMemo(() => (definitions.data ?? []).map(definition => definition.key), [definitions.data])
  const values = useQuery({
    queryKey: ['public-configs', keys],
    queryFn: () => api.getPublicConfigs(keys),
    enabled: keys.length > 0,
  })
  const [formValues, setFormValues] = useState<Record<string, string>>({})
  const resolvedValues = useMemo(() => {
    const nextValues: Record<string, string> = {}
    for (const definition of definitions.data ?? [])
      nextValues[definition.key] = formValues[definition.key] ?? values.data?.[definition.key] ?? definition.default
    return nextValues
  }, [definitions.data, formValues, values.data])

  const save = useMutation({
    mutationFn: api.updateConfigs,
    onSuccess: (result) => {
      toast.success('站点配置已保存')
      queryClient.setQueryData(['public-configs'], result)
      queryClient.invalidateQueries({ queryKey: ['public-configs'] })
    },
    onError: error => toast.error(error.message),
  })

  return (
    <div className="grid gap-6">
      <PageHeader
        description="这些配置是公开配置，前端会在启动后批量读取。"
        title="站点设置"
      />

      {definitions.isError && <ErrorState title="配置定义加载失败" description="请确认当前账号有权限访问站点设置。" />}

      <Card className="max-w-3xl">
        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            save.mutate(formValues)
          }}
        >
          {(definitions.data ?? []).map(definition => (
            <Field key={definition.key} label={definition.label}>
              <Input
                value={resolvedValues[definition.key] ?? ''}
                onChange={event => setFormValues(current => ({ ...current, [definition.key]: event.target.value }))}
              />
              <p className="text-xs font-normal text-muted-foreground">
                {definition.key}
                {' · '}
                {definition.description}
              </p>
            </Field>
          ))}

          <Button className="w-fit" disabled={save.isPending} type="submit">
            <Save size={16} />
            保存配置
          </Button>
        </form>
      </Card>
    </div>
  )
}
