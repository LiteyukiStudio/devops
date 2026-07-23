import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PaginationController } from './pagination'

describe('pagination controller', () => {
  it('does not render controls when the normalized page total is zero', () => {
    const { container } = render(<PaginationController initialPage={1} total={0} />)

    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })
})
