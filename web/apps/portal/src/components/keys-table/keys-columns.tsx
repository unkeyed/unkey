import type { ColumnDef, FilterFn, RowData, SortingFn } from "@tanstack/react-table";
import { MoreHorizontal, Pencil, RefreshCw, Trash2 } from "lucide-react";
import { UsageSparkline } from "~/components/keys-table/usage-sparkline";
import { Badge } from "~/components/ui/badge";
import { Button } from "~/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "~/components/ui/tooltip";
import type { Key } from "~/routes/dave-initial-design/-seed";

declare module "@tanstack/react-table" {
  // biome-ignore lint/correctness/noUnusedVariables: type-parameter names must match the upstream declaration for module augmentation
  interface ColumnMeta<TData extends RowData, TValue> {
    className?: string;
  }
}

type Status = "enabled" | "expired" | "disabled";

function getStatus(k: Key): Status {
  if (!k.enabled) {
    return "disabled";
  }
  if (k.expires && k.expires < Date.now()) {
    return "expired";
  }
  return "enabled";
}

export const globalSearchFn: FilterFn<Key> = (row, _id, needle) => {
  const q = String(needle ?? "")
    .trim()
    .toLowerCase();
  if (!q) {
    return true;
  }
  const k = row.original;
  return (k.name?.toLowerCase().includes(q) ?? false) || k.id.toLowerCase().includes(q);
};

const nameSortingFn: SortingFn<Key> = (a, b) => {
  const an = a.original.name;
  const bn = b.original.name;
  if (an === null && bn === null) {
    return 0;
  }
  if (an === null) {
    return 1;
  }
  if (bn === null) {
    return -1;
  }
  return an.localeCompare(bn);
};

type ColumnCallbacks = {
  onDelete?: (id: string) => void;
  onEdit?: (id: string) => void;
  onRotate?: (id: string) => void;
};

export function createKeysColumns({
  onDelete,
  onEdit,
  onRotate,
}: ColumnCallbacks): ColumnDef<Key>[] {
  return [
    {
      id: "name",
      accessorKey: "name",
      header: "ID",
      cell: ({ row }) => <KeyIdCell id={row.original.id} name={row.original.name} />,
      sortingFn: nameSortingFn,
      meta: { className: "w-40" },
    },
    {
      id: "start",
      accessorKey: "start",
      header: "Key",
      cell: ({ row }) => <StartPreview value={row.original.start} />,
      enableSorting: false,
      meta: { className: "w-44" },
    },
    {
      id: "usage",
      header: "Usage",
      cell: ({ row }) => (
        <UsageSparkline
          buckets={row.original.usage}
          errors={row.original.errors}
          ariaLabel={`${row.original.name ?? "Unnamed key"} usage, last 30 days`}
        />
      ),
      enableSorting: false,
      meta: { className: "w-48" },
    },
    {
      id: "status",
      accessorFn: (row) => getStatus(row),
      header: "Status",
      cell: ({ getValue }) => <StatusBadge status={getValue<Status>()} />,
      filterFn: (row, _id, value) => {
        if (!value || value === "all") {
          return true;
        }
        return row.getValue("status") === value;
      },
      meta: { className: "w-28" },
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => <span className="text-gray-11">{formatDate(row.original.createdAt)}</span>,
      meta: { className: "w-32" },
    },
    {
      id: "expires",
      accessorKey: "expires",
      header: "Expires",
      cell: ({ row }) =>
        row.original.expires ? (
          <span className="text-gray-11">{formatDate(row.original.expires)}</span>
        ) : (
          <span className="text-gray-9">Never</span>
        ),
      meta: { className: "w-32" },
    },
    {
      id: "actions",
      header: () => <span className="sr-only">Actions</span>,
      cell: ({ row }) => {
        const expired = getStatus(row.original) === "expired";
        return (
          <div className="text-center">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" aria-label="Open key actions">
                  <MoreHorizontal />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuGroup>
                  <DropdownMenuItem onSelect={() => onEdit?.(row.original.id)}>
                    <Pencil />
                    Edit key
                  </DropdownMenuItem>
                  {expired ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span>
                          <DropdownMenuItem disabled>
                            <RefreshCw />
                            Rotate key
                          </DropdownMenuItem>
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="left">Expired keys can't be rotated.</TooltipContent>
                    </Tooltip>
                  ) : (
                    <DropdownMenuItem onSelect={() => onRotate?.(row.original.id)}>
                      <RefreshCw />
                      Rotate key
                    </DropdownMenuItem>
                  )}
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuItem
                    variant="destructive"
                    onSelect={() => onDelete?.(row.original.id)}
                  >
                    <Trash2 />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
      enableSorting: false,
      meta: { className: "w-14" },
    },
  ];
}

function KeyIdCell({ id, name }: { id: string; name: string | null }) {
  return (
    <div className="flex flex-col gap-0.5 text-xs">
      {name ? (
        <>
          <span className="truncate font-medium text-gray-12" title={name}>
            {name}
          </span>
          <span className="truncate font-mono text-gray-11" title={id}>
            {id}
          </span>
        </>
      ) : (
        <span className="truncate font-medium font-mono text-gray-12" title={id}>
          {id}
        </span>
      )}
    </div>
  );
}

function StartPreview({ value }: { value: string }) {
  return <span className="font-mono text-gray-11 text-xs">{value.padEnd(16, "•")}</span>;
}

function StatusBadge({ status }: { status: Status }) {
  if (status === "enabled") {
    return <Badge variant="success">Enabled</Badge>;
  }
  if (status === "expired") {
    return <Badge variant="error">Expired</Badge>;
  }
  return <Badge variant="secondary">Disabled</Badge>;
}

function formatDate(ms: number) {
  return new Date(ms).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
