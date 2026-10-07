import type { CustomDomainDnsRecord } from "@/lib/collections/deploy/custom-domains";
import { IconCircleCheckOutline18 } from "@unkey/icons";
import { CopyButton, Skeleton } from "@unkey/ui";
import { cn } from "cn";

const DNS_RECORD_COLUMNS = "grid grid-cols-[72px_minmax(0,1fr)_minmax(0,1.5fr)] gap-3";

export function DnsRecordTable({
  records,
  verified,
}: {
  records: CustomDomainDnsRecord[];
  verified: boolean;
}) {
  const isLoading = records.length === 0;
  return (
    <div className="space-y-3">
      {isLoading ? (
        <Skeleton className="h-4 w-64 bg-gray-4 rounded" />
      ) : (
        <p className="text-xs text-gray-11">
          {verified
            ? "These DNS records point this domain at your app. Keep them in place."
            : "Add these records at your domain provider. If it offers a proxy, set them to DNS only."}
        </p>
      )}
      <div className="rounded-lg border bg-background overflow-hidden text-xs">
        <div
          className={cn(
            DNS_RECORD_COLUMNS,
            "bg-grayA-2 px-3 py-1.5 font-medium text-2xs text-gray-9 uppercase tracking-wider",
          )}
        >
          <span>Type</span>
          <span>Name</span>
          <span>Value</span>
        </div>
        <div className="divide-y">
          {isLoading ? (
            <>
              <DnsRecordRowSkeleton />
              <DnsRecordRowSkeleton />
            </>
          ) : (
            records.map((record) => (
              <DnsRecordRow key={`${record.type}:${record.name}`} record={record} />
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function DnsRecordRow({ record }: { record: CustomDomainDnsRecord }) {
  return (
    <div
      className={cn(DNS_RECORD_COLUMNS, "items-center px-3 py-2", record.verified && "text-gray-8")}
    >
      <span
        className={cn(
          "flex items-center gap-1 font-medium",
          record.verified ? "text-gray-8" : "text-gray-11",
        )}
      >
        {record.type}
        {record.verified && (
          <IconCircleCheckOutline18 aria-label="Verified" className="size-3! text-success-9" />
        )}
      </span>
      <DnsValue value={record.name} copyable={!record.verified} />
      <DnsValue value={record.value} copyable={!record.verified} />
    </div>
  );
}

function DnsValue({ value, copyable }: { value: string; copyable: boolean }) {
  return (
    <span className="flex min-w-0 items-center gap-1.5">
      <code className="min-w-0 truncate whitespace-nowrap! font-mono" title={value}>
        {value}
      </code>
      {copyable && <CopyButton value={value} className="size-5 shrink-0" variant="ghost" />}
    </span>
  );
}

function DnsRecordRowSkeleton() {
  return (
    <div className={cn(DNS_RECORD_COLUMNS, "items-center px-3 py-2")}>
      <Skeleton className="h-4 w-10 bg-gray-4 rounded" />
      <Skeleton className="h-4 w-32 bg-gray-4 rounded" />
      <Skeleton className="h-4 w-40 bg-gray-4 rounded" />
    </div>
  );
}
