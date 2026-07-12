import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DataList } from './data-list'

describe('data list action column', () => {
  it('uses content width even when a page supplies a legacy fixed width', () => {
    render(
      <DataList
        columns={[
          { key: 'name', header: 'Name', render: item => item.name },
          {
            key: 'actions',
            header: 'Actions',
            className: 'w-64 min-w-64 px-4 text-right',
            render: () => <button type="button">...</button>,
          },
        ]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One' }]}
        rowKey={item => item.id}
      />,
    )

    const header = screen.getByRole('columnheader', { name: 'Actions' })
    expect(header).toHaveClass('w-px', 'min-w-0', 'px-2', 'sm:px-4')
    expect(header).not.toHaveClass('w-64', 'min-w-64')
  })
})
