import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { DataTable, type DataTableColumn, type DataTableLabels } from './index';

interface Member {
  id: string;
  name: string;
  seats: number;
}

const columns: DataTableColumn<Member>[] = [
  { id: 'name', header: 'Name', cell: (row) => row.name, sortValue: (row) => row.name, rowHeader: true },
  { id: 'seats', header: 'Seats', cell: (row) => row.seats, sortValue: (row) => row.seats, align: 'end' },
  { id: 'id', header: 'Identifier', cell: (row) => row.id },
];

const rows: Member[] = [
  { id: 'm1', name: 'Charlie', seats: 3 },
  { id: 'm2', name: 'alpha', seats: 12 },
  { id: 'm3', name: 'Bravo', seats: 1 },
];

const labels: DataTableLabels = {
  pagination: 'Members pagination',
  previousPage: 'Previous',
  nextPage: 'Next',
  pageStatus: (page, pageCount) => `Page ${page} of ${pageCount}`,
};

function bodyNames() {
  const [, body] = screen.getAllByRole('rowgroup');
  return within(body).getAllByRole('rowheader').map((cell) => cell.textContent);
}

describe('DataTable block', () => {
  it('renders a captioned table with row headers from the column definitions', () => {
    render(
      <DataTable
        caption="Members"
        columns={columns}
        rows={rows}
        getRowId={(row) => row.id}
        labels={labels}
        emptyState="No members"
      />
    );

    expect(screen.getByRole('table', { name: 'Members' })).toBeInTheDocument();
    expect(bodyNames()).toEqual(['Charlie', 'alpha', 'Bravo']);
    expect(screen.getByRole('columnheader', { name: 'Identifier' })).not.toHaveAttribute('aria-sort');
  });

  it('sorts with header buttons and exposes the order through aria-sort', () => {
    const onSortChange = vi.fn();
    render(
      <DataTable
        caption="Members"
        columns={columns}
        rows={rows}
        getRowId={(row) => row.id}
        labels={labels}
        emptyState="No members"
        onSortChange={onSortChange}
      />
    );

    const nameHeader = screen.getByRole('columnheader', { name: 'Name' });
    expect(nameHeader).toHaveAttribute('aria-sort', 'none');

    fireEvent.click(within(nameHeader).getByRole('button', { name: 'Name' }));
    expect(nameHeader).toHaveAttribute('aria-sort', 'ascending');
    expect(bodyNames()).toEqual(['alpha', 'Bravo', 'Charlie']);

    fireEvent.click(within(nameHeader).getByRole('button', { name: 'Name' }));
    expect(nameHeader).toHaveAttribute('aria-sort', 'descending');
    expect(bodyNames()).toEqual(['Charlie', 'Bravo', 'alpha']);
    expect(onSortChange).toHaveBeenLastCalledWith({ columnId: 'name', direction: 'desc' });

    fireEvent.click(within(screen.getByRole('columnheader', { name: 'Seats' })).getByRole('button'));
    expect(bodyNames()).toEqual(['Bravo', 'Charlie', 'alpha']);
  });

  it('paginates with labelled controls and announces the page status', () => {
    render(
      <DataTable
        caption="Members"
        columns={columns}
        rows={rows}
        getRowId={(row) => row.id}
        labels={labels}
        emptyState="No members"
        pageSize={2}
      />
    );

    const navigation = screen.getByRole('navigation', { name: 'Members pagination' });
    expect(within(navigation).getByText('Page 1 of 2')).toHaveAttribute('aria-live', 'polite');
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByText('Page 2 of 2')).toBeInTheDocument();
    expect(bodyNames()).toEqual(['Bravo']);
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
  });

  it('renders the empty state across all columns without pagination', () => {
    render(
      <DataTable
        caption="Members"
        columns={columns}
        rows={[]}
        getRowId={(row) => row.id}
        labels={labels}
        emptyState={<p>No members yet</p>}
      />
    );

    expect(screen.getByText('No members yet').closest('td')).toHaveAttribute('colspan', '3');
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
  });
});
