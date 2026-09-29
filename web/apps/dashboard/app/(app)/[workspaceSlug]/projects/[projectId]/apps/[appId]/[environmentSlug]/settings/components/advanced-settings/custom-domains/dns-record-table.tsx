import type { CustomDomainDnsRecord } from "@/lib/collections/deploy/custom-domains";
import { Skeleton } from "@unkey/ui";
import { DnsRecordRow } from "./dns-record-row";

type DnsRecordTableProps = {
  records: CustomDomainDnsRecord[];
  isLoading?: boolean;
};

export function DnsRecordTable({ records, isLoading }: DnsRecordTableProps) {
  if (isLoading) {
    return (
      <div className="px-4 pb-3 space-y-3">
        <Skeleton className="h-4 w-64 bg-gray-4 rounded" />
        <div className="rounded-lg border bg-background overflow-hidden text-xs">
          <TableHeader />
          <DnsRecordRowSkeleton />
          <DnsRecordRowSkeleton isLast />
        </div>
      </div>
    );
  }

  return (
    <div className="px-4 pb-3 space-y-3">
      <p className="text-xs text-gray-9">Add the DNS records below at your domain provider.</p>

      <div className="rounded-lg border bg-background overflow-hidden text-xs">
        <TableHeader />
        {records.map((record, index) => (
          <DnsRecordRow
            key={`${record.type}:${record.name}`}
            type={record.type}
            name={record.name}
            value={record.value}
            verified={record.verified}
            isLast={index === records.length - 1}
          />
        ))}
      </div>

      {records.map(
        (record) =>
          record.note && (
            <p key={`${record.type}:${record.name}:note`} className="text-xs text-gray-9">
              <span className="font-medium">{record.type}</span> {record.note}
            </p>
          ),
      )}
    </div>
  );
}

function TableHeader() {
  return (
    <div className="grid grid-cols-[64px_1fr_1fr_48px] px-3 py-1.5 text-[11px] text-gray-9 font-medium uppercase tracking-wider bg-grayA-2">
      <span>Type</span>
      <span>Name</span>
      <span>Value</span>
      <span className="text-center">Status</span>
    </div>
  );
}

function DnsRecordRowSkeleton({ isLast }: { isLast?: boolean }) {
  return (
    <div
      className={`grid grid-cols-[64px_1fr_1fr_48px] px-3 py-2 items-center ${isLast ? "" : "border-b"}`}
    >
      <Skeleton className="h-4 w-10 bg-gray-4 rounded" />
      <Skeleton className="h-4 w-32 bg-gray-4 rounded" />
      <Skeleton className="h-4 w-40 bg-gray-4 rounded" />
      <div className="flex justify-center">
        <Skeleton className="size-3.5 bg-gray-4 rounded-full" />
      </div>
    </div>
  );
}
