import type { ColumnDef, FilterFn, RowData, SortingFn } from "@tanstack/react-table";
import { Check, Copy, MoreHorizontal, Pencil, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { UsageSparkline } from "~/components/keys-table/usage-sparkline";
import { Badge } from "~/components/ui/badge";
import { Button } from "~/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/components/ui/dropdown-menu";
import type { Key } from "~/routes/dave-initial-design/-seed";

declare module "@tanstack/react-table" {
  interface ColumnMeta<TData extends RowData, TValue> {
    className?: string;
  }
}

type Status = "enabled" | "expired" | "disabled";

function getStatus(k: Key): Status {
  if (!k.enabled) return "disabled";
  if (k.expires && k.expires < Date.now()) return "expired";
  return "enabled";
}

export const globalSearchFn: FilterFn<Key> = (row, _id, needle) => {
  const q = String(needle ?? "")
    .trim()
    .toLowerCase();
  if (!q) return true;
  const k = row.original;
  return (
    (k.name?.toLowerCase().includes(q) ?? false) ||
    k.id.toLowerCase().includes(q) ||
    k.externalId.toLowerCase().includes(q)
  );
};

const nameSortingFn: SortingFn<Key> = (a, b) => {
  const an = a.original.name;
  const bn = b.original.name;
  if (an === null && bn === null) return 0;
  if (an === null) return 1;
  if (bn === null) return -1;
  return an.localeCompare(bn);
};

type ColumnCallbacks = {
  onDelete?: (id: string) => void;
  onEditExpiration?: (id: string) => void;
  onRotate?: (id: string) => void;
};

export function createKeysColumns({
  onDelete,
  onEditExpiration,
  onRotate,
}: ColumnCallbacks): ColumnDef<Key>[] {
  return [
    {
      id: "name",
      accessorKey: "name",
      header: "ID",
      cell: ({ row }) => <KeyIdCell id={row.original.id} name={row.original.name} />,
      sortingFn: nameSortingFn,
    },
    {
      id: "start",
      accessorKey: "start",
      header: "Key",
      cell: ({ row }) => <CopyableStart value={row.original.start} />,
      enableSorting: false,
    },
    {
      id: "usage",
      header: "Usage",
      cell: ({ row }) => (
        <UsageSparkline
          buckets={row.original.usage}
          errors={row.original.errors}
          ariaLabel={`${row.original.name ?? "Unnamed key"} usage, last 30 hours`}
        />
      ),
      enableSorting: false,
    },
    {
      id: "status",
      accessorFn: (row) => getStatus(row),
      header: "Status",
      cell: ({ getValue }) => <StatusBadge status={getValue<Status>()} />,
      filterFn: (row, _id, value) => {
        if (!value || value === "all") return true;
        return row.getValue("status") === value;
      },
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => <span className="text-gray-11">{formatDate(row.original.createdAt)}</span>,
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
    },
    {
      id: "actions",
      header: () => <span className="sr-only">Actions</span>,
      cell: ({ row }) => (
        <div className="text-right">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" aria-label="Open key actions">
                <MoreHorizontal />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => onEditExpiration?.(row.original.id)}>
                <Pencil />
                Edit expiration
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => onRotate?.(row.original.id)}>
                <RefreshCw />
                Rotate
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onSelect={() => onDelete?.(row.original.id)}>
                <Trash2 />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      ),
      enableSorting: false,
      meta: { className: "w-10" },
    },
  ];
}

function KeyIdCell({ id, name }: { id: string; name: string | null }) {
  return (
    <div className="flex flex-col gap-0.5 text-xs">
      {name ? (
        <>
          <span className="max-w-40 truncate font-medium text-gray-12" title={name}>
            {name}
          </span>
          <span className="font-mono text-gray-11">{id}</span>
        </>
      ) : (
        <span className="font-medium font-mono text-gray-12">{id}</span>
      )}
    </div>
  );
}

function CopyableStart({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const display = value.padEnd(16, "•");

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      title="Click to copy"
      className="group inline-flex items-center gap-2 rounded px-1.5 py-0.5 font-mono text-gray-11 text-xs transition-colors hover:bg-gray-2 hover:text-gray-12"
    >
      <span>{display}</span>
      {copied ? (
        <Check className="size-3 text-successA-11" />
      ) : (
        <Copy className="size-3 opacity-0 transition-opacity group-hover:opacity-100" />
      )}
    </button>
  );
}

function StatusBadge({ status }: { status: Status }) {
  if (status === "enabled") return <Badge variant="success">Enabled</Badge>;
  if (status === "expired") return <Badge variant="error">Expired</Badge>;
  return <Badge variant="secondary">Disabled</Badge>;
}

function formatDate(ms: number) {
  return new Date(ms).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
