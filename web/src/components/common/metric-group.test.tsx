import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { MetricGroup, MetricItem } from './metric-group'

describe('metric group semantics', () => {
  it('maps risk tones to semantic surfaces and values', () => {
    render(
      <MemoryRouter>
        <MetricGroup>
          <MetricItem href="/events?status=failed" label="Failed" tone="danger" value="3" />
          <MetricItem emphasis={false} label="No pending work" value="0" />
        </MetricGroup>
      </MemoryRouter>,
    )

    const failed = screen.getByRole('link', { name: /Failed/ })
    expect(failed).toHaveClass('bg-danger-subtle/45', 'ring-danger-border/45')
    expect(screen.getByText('3')).toHaveClass('text-danger')
    expect(screen.getByText('0')).toHaveClass('text-muted-foreground')
  })
})
