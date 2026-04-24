type Props = {
  buckets: number[];
  errors?: number[];
};

export function UsagePopover({ buckets, errors }: Props) {
  const errorTotal = (errors ?? []).reduce((acc, n) => acc + n, 0);
  const total = buckets.reduce((acc, n) => acc + n, 0);
  const validTotal = Math.max(total - errorTotal, 0);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col">
        <span className="font-medium text-gray-12 text-sm">API Key Activity</span>
        <span className="text-gray-11 text-xs">Last 30 hours</span>
      </div>
      <div className="flex flex-col gap-1 text-xs">
        <Row label="Valid" count={validTotal} swatch="bg-gray-7" />
        {errorTotal > 0 && <Row label="Errors" count={errorTotal} swatch="bg-error-9" />}
      </div>
    </div>
  );
}

function Row({ label, count, swatch }: { label: string; count: number; swatch: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="flex items-center gap-2 text-gray-11">
        <span className={`size-1.5 rounded-[1px] ${swatch}`} aria-hidden />
        {label}
      </span>
      <span className="text-gray-12 tabular-nums">{count.toLocaleString()}</span>
    </div>
  );
}
