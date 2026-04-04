import { getDomain } from "tldts";
import { DnsRecordRow } from "./dns-record-row";

type DnsRecordTableProps = {
  domain: string;
  targetCname: string;
  verificationToken: string;
  ownershipVerified: boolean;
  cnameVerified: boolean;
  isLoading?: boolean;
};

/** Returns true for apex/root domains using the public suffix list via tldts. */
function isApexDomain(domain: string): boolean {
  const registrable = getDomain(domain);
  return registrable === domain;
}

function DnsRecordRowSkeleton({ isLast }: { isLast?: boolean }) {
  return (
    <div
      className={`grid grid-cols-[64px_1fr_1fr_48px] px-3 py-2 items-center ${isLast ? "" : "border-b border-gray-3"}`}
    >
      <div className="h-4 w-10 bg-gray-4 rounded animate-pulse" />
      <div className="h-4 w-32 bg-gray-4 rounded animate-pulse" />
      <div className="h-4 w-40 bg-gray-4 rounded animate-pulse" />
      <div className="flex justify-center">
        <div className="size-3.5 bg-gray-4 rounded-full animate-pulse" />
      </div>
    </div>
  );
}

export function DnsRecordTable({
  domain,
  targetCname,
  verificationToken,
  ownershipVerified,
  cnameVerified,
  isLoading,
}: DnsRecordTableProps) {
  const apex = isApexDomain(domain);
  const recordType = apex ? "ALIAS" : "CNAME";

  // TXT record is only needed for apex domains (CNAME flattening hides the CNAME)
  // or when CNAME verification failed (proxy, contested, etc.)
  const showTxtRecord = apex || (!cnameVerified && ownershipVerified === false);

  const helpText = apex
    ? "Add both DNS records below. For the ALIAS record, use an ALIAS, ANAME, or flattened CNAME (Cloudflare) depending on your provider. The TXT record is required for apex domains."
    : showTxtRecord
      ? "Add both DNS records below at your domain provider."
      : "Add the CNAME record below at your domain provider.";

  const txtRecordName = `_unkey.${domain}`;
  const txtRecordValue = `unkey-domain-verify=${verificationToken}`;

  if (isLoading) {
    return (
      <div className="px-4 pb-3 space-y-3">
        <div className="h-4 w-64 bg-gray-4 rounded animate-pulse" />
        <div className="rounded-lg border border-gray-4 overflow-hidden text-xs">
          <div className="grid grid-cols-[64px_1fr_1fr_48px] px-3 py-1.5 text-[11px] text-gray-9 font-medium uppercase tracking-wider bg-grayA-2">
            <span>Type</span>
            <span>Name</span>
            <span>Value</span>
            <span className="text-center">Status</span>
          </div>
          <DnsRecordRowSkeleton />
          <DnsRecordRowSkeleton isLast />
        </div>
      </div>
    );
  }

  return (
    <div className="px-4 pb-3 space-y-3">
      <p className="text-xs text-gray-9">{helpText}</p>

      <div className="rounded-lg border border-gray-4 overflow-hidden text-xs">
        <div className="grid grid-cols-[64px_1fr_1fr_48px] px-3 py-1.5 text-[11px] text-gray-9 font-medium uppercase tracking-wider bg-grayA-2">
          <span>Type</span>
          <span>Name</span>
          <span>Value</span>
          <span className="text-center">Status</span>
        </div>

        {showTxtRecord && (
          <DnsRecordRow
            type="TXT"
            name={txtRecordName}
            value={txtRecordValue}
            verified={ownershipVerified}
          />
        )}
        <DnsRecordRow
          type={recordType}
          name={domain}
          value={targetCname}
          verified={cnameVerified}
          isLast
        />
      </div>
    </div>
  );
}
