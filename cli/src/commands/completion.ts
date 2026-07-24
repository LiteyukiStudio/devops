import type { CommandRegistry } from './registry.js'

export type CompletionShell = 'bash' | 'zsh' | 'fish' | 'powershell'

export function generateCompletion(shell: CompletionShell, registry: CommandRegistry): string {
  const categories = registry.categories()
  const toolsByCategory = Object.fromEntries(
    categories.map(category => [
      category,
      registry
        .list({ category, includeHidden: false })
        .map(command => command.metadata.tool),
    ]),
  )

  switch (shell) {
    case 'bash':
      return bashCompletion(categories, toolsByCategory)
    case 'zsh':
      return zshCompletion(categories, toolsByCategory)
    case 'fish':
      return fishCompletion(categories, toolsByCategory)
    case 'powershell':
      return powershellCompletion(categories, toolsByCategory)
  }
}

function bashCompletion(
  categories: readonly string[],
  tools: Readonly<Record<string, readonly string[]>>,
): string {
  const cases = Object.entries(tools)
    .map(([category, items]) => `      ${category}) COMPREPLY=( $(compgen -W "${items.join(' ')}" -- "$cur") ) ;;`)
    .join('\n')
  return `# Luna CLI completion (generated)
_luna_completion() {
  local cur category
  cur="\${COMP_WORDS[COMP_CWORD]}"
  category="\${COMP_WORDS[1]}"
  if [[ \${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "${categories.join(' ')}" -- "$cur") )
    return
  fi
  if [[ \${COMP_CWORD} -eq 2 ]]; then
    case "$category" in
${cases}
    esac
  fi
}
complete -F _luna_completion luna
`
}

function zshCompletion(
  categories: readonly string[],
  tools: Readonly<Record<string, readonly string[]>>,
): string {
  const cases = Object.entries(tools)
    .map(([category, items]) => `    ${category}) _values 'tool' ${items.join(' ')} ;;`)
    .join('\n')
  return `#compdef luna
_luna() {
  local category
  if (( CURRENT == 2 )); then
    _values 'category' ${categories.join(' ')}
    return
  fi
  category="$words[2]"
  if (( CURRENT == 3 )); then
    case "$category" in
${cases}
    esac
  fi
}
_luna "$@"
`
}

function fishCompletion(
  categories: readonly string[],
  tools: Readonly<Record<string, readonly string[]>>,
): string {
  const categoryLines = categories
    .map(category => `complete -c luna -n '__fish_use_subcommand' -a '${category}'`)
    .join('\n')
  const toolLines = Object.entries(tools)
    .flatMap(([category, items]) =>
      items.map(
        tool =>
          `complete -c luna -n '__fish_seen_subcommand_from ${category}' -a '${tool}'`,
      ),
    )
    .join('\n')
  return `# Luna CLI completion (generated)
${categoryLines}
${toolLines}
`
}

function powershellCompletion(
  categories: readonly string[],
  tools: Readonly<Record<string, readonly string[]>>,
): string {
  const serialized = JSON.stringify(tools)
  return `# Luna CLI completion (generated)
Register-ArgumentCompleter -Native -CommandName luna -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  $elements = @($commandAst.CommandElements | ForEach-Object { $_.Value })
  $tools = ConvertFrom-Json '${serialized}' -AsHashtable
  $candidates = if ($elements.Count -le 2) {
    @(${categories.map(value => `'${value}'`).join(', ')})
  } elseif ($elements.Count -eq 3 -and $tools.ContainsKey($elements[1])) {
    @($tools[$elements[1]])
  } else {
    @()
  }
  $candidates | Where-Object { $_ -like "$wordToComplete*" } |
    ForEach-Object { [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_) }
}
`
}
