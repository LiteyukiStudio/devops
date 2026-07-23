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
    expect(screen.getByRole('cell', { name: '...' })).toHaveClass('sticky', 'right-0', 'border-separator-strong')
  })

  it('renders query controls in the list header toolbar', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One' }]}
        rowKey={item => item.id}
        title="Projects"
        toolbar={<button type="button">Sort projects</button>}
      />,
    )

    const toolbarButton = screen.getByRole('button', { name: 'Sort projects' })
    expect(screen.getByText('Projects').parentElement?.parentElement).toContainElement(toolbarButton)
  })

  it('uses a clean white header surface and left-aligns query controls without a repeated title', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One' }]}
        rowKey={item => item.id}
        search={{ value: '', placeholder: 'Search projects', onChange: () => undefined }}
        toolbar={<button type="button">Sort projects</button>}
      />,
    )

    expect(screen.getAllByRole('rowgroup')[0]).toHaveClass('bg-card/95')
    expect(screen.getAllByRole('rowgroup')[0]).not.toHaveClass('border-separator-strong')
    expect(screen.getByRole('row', { name: 'One' })).toHaveClass(
      'border-separator-strong',
      'border-t',
      'hover:border-surface-subtle',
      'hover:[&>td]:bg-surface-subtle',
      '[&>td:first-child]:rounded-l-container',
      '[&>td:last-child]:rounded-r-container',
    )
    expect(screen.getByRole('button', { name: 'Sort projects' }).closest('[data-slot="data-list-tools"]')).toHaveClass(
      'after:border-separator-strong',
      'after:inset-x-4',
    )
    expect(screen.getByRole('table').closest('[data-slot="scroll-area"]')).toHaveClass('mx-group')
    const search = screen.getByPlaceholderText('Search projects')
    expect(search.parentElement).not.toHaveClass('sm:justify-end')
    expect(search.parentElement).toContainElement(screen.getByRole('button', { name: 'Sort projects' }))
    expect(screen.getByRole('table').closest('[data-slot="card"]')).not.toHaveClass('p-section')
  })

  it('only draws a toolbar separator when tools are present', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyTitle="Empty"
        items={[{ id: 'one', name: 'One' }]}
        pagination={{
          page: 1,
          pageInfoLabel: '1 item',
          pageSize: 10,
          total: 1,
          totalPages: 1,
          onPageChange: () => undefined,
        }}
        rowKey={item => item.id}
        title="Projects"
      />,
    )

    const titleBar = screen.getByText('Projects').parentElement?.parentElement
    const pagination = screen.getByText('1 item').parentElement?.parentElement
    expect(titleBar?.className).not.toContain('after:')
    expect(pagination?.className).not.toContain('before:')
    expect(screen.getByRole('row', { name: 'One' })).toHaveClass('border-t', 'border-separator-strong')
  })

  it('uses the same top border for the header-to-row and row-to-row separators', () => {
    render(
      <DataList
        columns={[{ key: 'name', header: 'Name', render: item => item.name }]}
        emptyTitle="Empty"
        items={[
          { id: 'one', name: 'One' },
          { id: 'two', name: 'Two' },
        ]}
        rowKey={item => item.id}
      />,
    )

    const rows = screen.getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)
    for (const row of rows)
      expect(row).toHaveClass('border-t', 'border-separator-strong')
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
