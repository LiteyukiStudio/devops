import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { PageChrome } from './page-chrome'

describe('page chrome', () => {
  it('keeps the title row height stable when tools are absent', () => {
    const { container } = render(
      <MemoryRouter>
        <PageChrome
          tabsTargetRef={() => undefined}
          title={<h1>Projects</h1>}
          toolsTargetRef={() => undefined}
        />
      </MemoryRouter>,
    )

    expect(container.querySelector('[data-slot="page-chrome-title-row"]')).toHaveClass('min-h-10')
  })

  it('places optional back navigation below the title', () => {
    const { container } = render(
      <MemoryRouter>
        <PageChrome
          backNavigation={{ label: 'Back to project spaces', to: '/projects' }}
          tabsTargetRef={() => undefined}
          title={<h1>Project workspace</h1>}
          toolsTargetRef={() => undefined}
        />
      </MemoryRouter>,
    )

    const titleRow = container.querySelector('[data-slot="page-chrome-title-row"]')
    const desktopBackLink = container.querySelector('.lg\\:flex [data-slot="page-chrome-back-navigation"]')
    const tabsRow = container.querySelector('[data-slot="page-chrome-tabs-row"]')

    expect(screen.getAllByRole('link', { name: 'Back to project spaces' })).toHaveLength(2)
    expect(desktopBackLink).toHaveAttribute('href', '/projects')
    expect(titleRow!.compareDocumentPosition(desktopBackLink!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(desktopBackLink!.compareDocumentPosition(tabsRow!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })
})
