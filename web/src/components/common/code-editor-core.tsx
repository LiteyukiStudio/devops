import type { CodeEditorProps } from './code-editor'
import { json } from '@codemirror/lang-json'
import { yaml } from '@codemirror/lang-yaml'
import CodeMirror from '@uiw/react-codemirror'
import { useEffect, useMemo, useState } from 'react'
import { cn } from '@/lib/utils'

type CodeEditorLanguage = NonNullable<CodeEditorProps['language']>

export function CodeEditorCore({
  value,
  onChange,
  language = 'text',
  height,
  minHeight = '14rem',
  placeholder,
  readOnly,
  className,
  ariaInvalid,
}: CodeEditorProps) {
  const [dark, setDark] = useState(() => document.documentElement.dataset.theme === 'dark')
  const extensions = useMemo(() => languageExtensions(language), [language])

  useEffect(() => {
    const observer = new MutationObserver(() => {
      setDark(document.documentElement.dataset.theme === 'dark')
    })
    observer.observe(document.documentElement, { attributeFilter: ['data-theme'] })
    return () => observer.disconnect()
  }, [])

  return (
    <div
      className={cn(
        'liteyuki-code-editor min-w-0 max-w-full overflow-hidden rounded-md border border-input bg-surface shadow-xs transition-[border-color,box-shadow]',
        'focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50',
        ariaInvalid && 'border-destructive ring-destructive/20 dark:ring-destructive/40',
        className,
      )}
    >
      <div className="flex h-8 items-center justify-between border-b border-border bg-muted/70 px-3">
        <span className="font-mono text-[11px] font-medium uppercase tracking-normal text-muted-foreground">{language}</span>
      </div>
      <CodeMirror
        basicSetup={{
          bracketMatching: true,
          closeBrackets: true,
          foldGutter: false,
          highlightActiveLine: true,
          highlightActiveLineGutter: false,
          lineNumbers: true,
        }}
        editable={!readOnly}
        extensions={extensions}
        height={height ?? 'auto'}
        minHeight={height ? undefined : minHeight}
        placeholder={placeholder}
        readOnly={readOnly}
        style={{ maxWidth: '100%', minWidth: 0, width: '100%' }}
        theme={dark ? 'dark' : 'light'}
        value={value}
        onChange={onChange}
      />
    </div>
  )
}

function languageExtensions(language: CodeEditorLanguage) {
  switch (language) {
    case 'json':
      return [json()]
    case 'yaml':
      return [yaml()]
    default:
      return []
  }
}
