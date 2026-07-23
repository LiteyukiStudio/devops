import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import i18next from '@/i18n'
import { SearchMultiSelect } from './search-select'

describe('search multi select sizes', () => {
  beforeEach(async () => {
    await i18next.changeLanguage('zh-CN')
  })

  it('applies compact sizing to the trigger, search input, and options', () => {
    render(
      <SearchMultiSelect
        options={[
          { label: 'Build started', value: 'build_started', description: 'Build' },
          { label: 'Build succeeded', value: 'build_succeeded', description: 'Build' },
        ]}
        placeholder="All event types"
        size="sm"
        value={[]}
        onValueChange={vi.fn()}
      />,
    )

    const trigger = screen.getByRole('button', { name: 'All event types' })
    expect(trigger).toHaveClass('h-8', 'text-xs')

    fireEvent.click(trigger)

    expect(screen.getByPlaceholderText('搜索')).toHaveClass('h-7', 'text-xs')
    expect(screen.getByRole('button', { name: /Build started/ })).toHaveClass('py-1.5', 'text-xs')
  })
})
