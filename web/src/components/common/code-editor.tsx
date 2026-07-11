import { lazy, Suspense } from 'react'
import { cn } from '@/lib/utils'

type CodeEditorLanguage = 'json' | 'yaml' | 'text'
export interface CodeEditorProps {
  value: string
  onChange: (value: string) => void
  language?: CodeEditorLanguage
  height?: string
  minHeight?: string
  placeholder?: string
  readOnly?: boolean
  className?: string
  ariaInvalid?: boolean
}

const CodeEditorCore = lazy(() => import('./code-editor-core').then(module => ({ default: module.CodeEditorCore })))

/**
 * 代码编辑输入框。
 * 用于 kubeconfig、JSON 配置、YAML 清单、脚本片段等需要等宽字体、语法高亮和滚动编辑的长文本字段；普通备注或描述仍使用 Textarea。
 */
export function CodeEditor(props: CodeEditorProps) {
  return (
    <Suspense fallback={<CodeEditorFallback {...props} />}>
      <CodeEditorCore {...props} />
    </Suspense>
  )
}

function CodeEditorFallback({
  value,
  language = 'text',
  height,
  minHeight = '14rem',
  placeholder,
  className,
  ariaInvalid,
}: CodeEditorProps) {
  return (
    <div
      className={cn(
        'luna-devops-code-editor min-w-0 max-w-full overflow-hidden rounded-md border border-input bg-surface shadow-xs transition-[border-color,box-shadow]',
        ariaInvalid && 'border-destructive ring-destructive/20 dark:ring-destructive/40',
        className,
      )}
      style={{ minHeight: height ? undefined : minHeight }}
    >
      <div className="flex h-8 items-center justify-between border-b border-border bg-muted/70 px-3">
        <span className="font-mono text-[11px] font-medium uppercase tracking-normal text-muted-foreground">{language}</span>
      </div>
      <textarea
        className="block w-full resize-none bg-transparent p-3 font-mono text-sm outline-none placeholder:text-muted-foreground"
        placeholder={placeholder}
        readOnly
        style={{ height: height ?? `calc(${minHeight} - 2rem)`, minHeight: height ? undefined : `calc(${minHeight} - 2rem)` }}
        value={value}
      />
    </div>
  )
}
