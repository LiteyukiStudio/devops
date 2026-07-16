import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DataList } from './data-list'

describe('data list layout', () => {
  it('keeps intrinsic column width so a narrow container can scroll horizontally', () => {
    render(
      <DataList
        columns={[
          { key: 'name', header: 'Name', minWidth: 320, render: item => item.name },
          { key: 'description', header: 'Description', minWidth: 480, render: item => item.description },
        ]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One', description: 'Description' }]}
        rowKey={item => item.id}
      />,
    )

    const table = screen.getByRole('table')
    expect(table).toHaveClass('w-max', 'min-w-full')
    expect(table.closest('[data-scrollbars="both"]')).toHaveAttribute('data-scroll-area-type', 'auto')
  })

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

  it('keeps explicitly sticky action columns fixed on the right', () => {
    render(
      <DataList
        columns={[
          { key: 'name', header: 'Name', render: item => item.name },
          {
            key: 'actions',
            header: 'Actions',
            sticky: 'right',
            render: () => <button type="button">...</button>,
          },
        ]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One' }]}
        rowKey={item => item.id}
      />,
    )

    expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass('sticky', 'right-0')
    expect(screen.getByRole('cell', { name: '...' })).toHaveClass('sticky', 'right-0')
  })
})
