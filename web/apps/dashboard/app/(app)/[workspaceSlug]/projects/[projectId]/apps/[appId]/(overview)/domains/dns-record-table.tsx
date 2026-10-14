import type { CustomDomainDnsRecord } from "@/lib/collections/deploy/custom-domains";
import { IconCircleCheckOutline18 } from "@unkey/icons";
import { CopyButton } from "@unkey/ui";
import { cn } from "cn";

export function DnsRecordTable({
  records,
  verified,
}: {
  records: CustomDomainDnsRecord[];
  verified: boolean;
}) {
  return (
    <div className="space-y-3 [--dns-columns:72px_minmax(0,1fr)_minmax(0,1.5fr)]">
      <p className="text-xs text-gray-11">
        {verified
          ? "These DNS records point this domain at your app. Keep them in place."
          : "Add these records at your domain provider. If it offers a proxy, set them to DNS only."}
      </p>
      <div className="rounded-lg border bg-background overflow-hidden text-xs">
        <div className="grid grid-cols-(--dns-columns) gap-3 bg-grayA-2 px-3 py-1.5 font-medium text-2xs text-gray-9 uppercase tracking-wider">
          <span>Type</span>
          <span>Name</span>
          <span>Value</span>
        </div>
        <div className="divide-y">
          {records.length > 0 ? (
            records.map((record) => (
              <DnsRecordRow key={`${record.type}:${record.name}`} record={record} />
            ))
          ) : (
            <p className="px-3 py-2 text-gray-9">No DNS records yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}

function DnsRecordRow({ record }: { record: CustomDomainDnsRecord }) {
  return (
    <div
      className={cn(
        "grid grid-cols-(--dns-columns) items-center gap-3 px-3 py-2",
        record.verified && "text-gray-8",
      )}
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
