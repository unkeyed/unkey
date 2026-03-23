import { DnsRecordRow } from "./dns-record-row";

type DnsRecordTableProps = {
  domain: string;
  targetCname: string;
  verificationToken: string;
  ownershipVerified: boolean;
  cnameVerified: boolean;
};

export function DnsRecordTable({
  domain,
  targetCname,
  verificationToken,
  ownershipVerified,
  cnameVerified,
}: DnsRecordTableProps) {
  const txtRecordName = `_unkey.${domain}`;
  const txtRecordValue = `unkey-domain-verify=${verificationToken}`;

  return (
    <div className="px-4 pb-3 space-y-3">
      <p className="text-xs text-gray-9">Add both DNS records below at your domain provider.</p>

      <div className="rounded-lg border border-gray-4 overflow-hidden text-xs">
        <div className="grid grid-cols-[64px_1fr_1fr_48px] px-3 py-1.5 text-[11px] text-gray-9 font-medium uppercase tracking-wider bg-grayA-2">
          <span>Type</span>
          <span>Name</span>
          <span>Value</span>
          <span className="text-center">Status</span>
        </div>

        <DnsRecordRow
          type="TXT"
          name={txtRecordName}
          value={txtRecordValue}
          verified={ownershipVerified}
        />
        <DnsRecordRow
          type="CNAME"
          name={domain}
          value={targetCname}
          verified={cnameVerified}
          isLast
        />
      </div>
    </div>
  );
}
