import { DnsRecordRow } from "./dns-record-row";

type DnsRecordTableProps = {
  domain: string;
  targetCname: string;
  verificationToken: string;
  ownershipVerified: boolean;
  cnameVerified: boolean;
};

/** Returns true for apex/root domains (e.g. "example.com" but not "api.example.com"). */
function isApexDomain(domain: string): boolean {
  return domain.split(".").length === 2;
}

export function DnsRecordTable({
  domain,
  targetCname,
  verificationToken,
  ownershipVerified,
  cnameVerified,
}: DnsRecordTableProps) {
  const apex = isApexDomain(domain);
  const recordType = apex ? "ALIAS" : "CNAME";

  const helpText = apex
    ? "Add both DNS records below. For the ALIAS record, use an ALIAS, ANAME, or flattened CNAME (Cloudflare) depending on your provider. The TXT record is required for apex domains."
    : "Add both DNS records below at your domain provider. For subdomains, the CNAME record alone is sufficient — the TXT record is only needed if the CNAME cannot be verified (e.g. when using a proxy).";

  const txtRecordName = `_unkey.${domain}`;
  const txtRecordValue = `unkey-domain-verify=${verificationToken}`;

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

        <DnsRecordRow
          type="TXT"
          name={txtRecordName}
          value={txtRecordValue}
          verified={ownershipVerified}
        />
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
