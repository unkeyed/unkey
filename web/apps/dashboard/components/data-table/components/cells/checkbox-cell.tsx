import type { Row, Table } from "@tanstack/react-table";
import { Checkbox } from "@unkey/ui";

interface CheckboxCellProps<TData> {
  row: Row<TData>;
}

/**
 * Checkbox cell for row selection
 * Supports individual row selection and indeterminate state
 */
export function CheckboxCell<TData>({ row }: CheckboxCellProps<TData>) {
  return (
    <div className="flex items-center">
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
      />
    </div>
  );
}

interface CheckboxHeaderCellProps<TData> {
  table: Table<TData>;
}

/**
 * Checkbox header cell for select-all functionality
 */
export function CheckboxHeaderCell<TData>({ table }: CheckboxHeaderCellProps<TData>) {
  return (
    <div className="flex items-center">
      <Checkbox
        checked={table.getIsAllRowsSelected()}
        onCheckedChange={(value) => table.toggleAllRowsSelected(!!value)}
        aria-label="Select all rows"
      />
    </div>
  );
}
