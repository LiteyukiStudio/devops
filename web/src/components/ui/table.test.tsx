import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './table'

describe('table layout', () => {
  it('preserves intrinsic width inside its horizontal scroll container', () => {
    render(
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Description</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow>
            <TableCell>Example</TableCell>
            <TableCell>Details</TableCell>
          </TableRow>
        </TableBody>
      </Table>,
    )

    const table = screen.getByRole('table')
    expect(table).toHaveClass('w-max', 'min-w-full')
    expect(table.closest('[data-slot="table-container"]')).toHaveAttribute('data-scroll-area-type', 'auto')
    expect(table.closest('[data-slot="table-container"]')).toHaveAttribute('data-scrollbars', 'horizontal')
  })
})
