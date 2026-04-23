import { Check, Copy, KeyRound, MoreHorizontal, Pencil, RefreshCw, Trash2 } from "lucide-react";
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
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "~/components/ui/empty";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "~/components/ui/table";
import type { Key } from "~/routes/dave-initial-design/-seed";

type Props = {
  keys: Key[];
  onCreate?: () => void;
  onDelete?: (id: string) => void;
  onEditExpiration?: (id: string) => void;
  onRotate?: (id: string) => void;
};

export function KeysTable({ keys, onCreate, onDelete, onEditExpiration, onRotate }: Props) {
  return (
    <section className="flex flex-col gap-4">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-12">API Keys</h1>
          <p className="text-sm text-gray-11">{keys.length} keys</p>
        </div>
        <Button onClick={onCreate}>+ Create key</Button>
      </header>

      <div className="overflow-hidden rounded-lg border border-primary/10 bg-background shadow-xs">
        {keys.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <KeyRound />
              </EmptyMedia>
              <EmptyTitle>No API keys yet</EmptyTitle>
              <EmptyDescription>
                Create your first key to start making authenticated requests.
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button onClick={onCreate}>+ Create key</Button>
            </EmptyContent>
          </Empty>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-grayA-2 hover:bg-grayA-2">
                <TableHead>Key</TableHead>
                <TableHead>Value</TableHead>
                <TableHead>Usage</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead className="w-10 sr-only">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((k) => (
                <KeyRow
                  key={k.id}
                  k={k}
                  onDelete={onDelete}
                  onEditExpiration={onEditExpiration}
                  onRotate={onRotate}
                />
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </section>
  );
}

function KeyRow({
  k,
  onDelete,
  onEditExpiration,
  onRotate,
}: {
  k: Key;
  onDelete?: (id: string) => void;
  onEditExpiration?: (id: string) => void;
  onRotate?: (id: string) => void;
}) {
  const status = getStatus(k);
  return (
    <TableRow className="h-14">
      <TableCell>
        <KeyIdCell id={k.id} name={k.name} />
      </TableCell>
      <TableCell>
        <CopyableStart value={k.start} />
      </TableCell>
      <TableCell>
        <UsageSparkline
          buckets={k.usage}
          errors={k.errors}
          ariaLabel={`${k.name ?? "Unnamed key"} usage, last 30 hours`}
        />
      </TableCell>
      <TableCell>
        <StatusBadge status={status} />
      </TableCell>
      <TableCell className="text-gray-11">{formatDate(k.createdAt)}</TableCell>
      <TableCell className="text-gray-11">
        {k.expires ? formatDate(k.expires) : <span className="text-gray-9">Never</span>}
      </TableCell>
      <TableCell className="text-right">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" aria-label="Open key actions">
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onSelect={() => onEditExpiration?.(k.id)}>
              <Pencil />
              Edit expiration
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => onRotate?.(k.id)}>
              <RefreshCw />
              Rotate
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onSelect={() => onDelete?.(k.id)}>
              <Trash2 />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
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
        <span className="font-mono font-medium text-gray-12">{id}</span>
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
      className="group inline-flex items-center gap-2 rounded px-1.5 py-0.5 font-mono text-xs text-gray-11 transition-colors hover:bg-grayA-2 hover:text-gray-12"
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
