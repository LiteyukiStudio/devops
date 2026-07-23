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

  it('does not render pagination controls for an empty result', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyTitle="Empty"
        items={[] as { id: string, name: string }[]}
        pagination={{
          page: 1,
          pageInfoLabel: '0 items',
          pageSize: 10,
          total: 0,
          totalPages: 0,
          onPageChange: () => undefined,
        }}
        rowKey={item => item.id}
      />,
    )

    expect(screen.queryByText('0 items')).not.toBeInTheDocument()
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })

  it('renders a structured loading state instead of the empty state', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyTitle="Empty"
        items={[] as { id: string, name: string }[]}
        loading
        rowKey={item => item.id}
      />,
    )

    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true')
    expect(screen.queryByText('Empty')).not.toBeInTheDocument()
  })

  it('can hide secondary columns on mobile while keeping action columns intact', () => {
    render(
      <DataList
        columns={[
          { key: 'name', header: 'Name', render: item => item.name },
          { key: 'detail', header: 'Detail', mobile: 'hidden', render: item => item.detail },
          { key: 'actions', header: 'Actions', sticky: 'right', render: () => <button type="button">Open</button> },
        ]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One', detail: 'Secondary' }]}
        rowKey={item => item.id}
      />,
    )

    expect(screen.getByRole('columnheader', { name: 'Detail' })).toHaveClass('hidden', 'md:table-cell')
    expect(screen.getByRole('columnheader', { name: 'Actions' })).not.toHaveClass('hidden')
  })

  it('renders filtered empty results as a compact centered state with a clear action', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyActions={<button type="button">Clear filters</button>}
        emptyMode="filtered"
        emptyTitle="No matching results"
        items={[] as { id: string, name: string }[]}
        rowKey={item => item.id}
      />,
    )

    expect(screen.getByText('No matching results').closest('[data-slot="empty"]')).toHaveClass('min-h-24', 'items-center')
    expect(screen.getByRole('button', { name: 'Clear filters' })).toBeInTheDocument()
  })
})
