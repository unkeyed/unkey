import { Meter, MeterHeader, MeterIndicator, MeterLabel, MeterTrack, MeterValue } from "@unkey/ui";

const QUOTA = 150_000;

function indicatorColor(percent: number) {
  if (percent >= 100) {
    return "bg-error-9";
  }
  if (percent >= 80) {
    return "bg-warning-9";
  }
  return "bg-info-9";
}

function QuotaMeter({ label, used }: { label: string; used: number }) {
  const percent = (used / QUOTA) * 100;

  return (
    <Meter value={used} max={QUOTA}>
      <MeterHeader>
        <MeterLabel>{label}</MeterLabel>
        <MeterValue>
          {used.toLocaleString()} / {QUOTA.toLocaleString()}
        </MeterValue>
      </MeterHeader>
      <MeterTrack>
        <MeterIndicator className={indicatorColor(percent)} />
      </MeterTrack>
    </Meter>
  );
}

export function ThresholdMeter() {
  return (
    <div className="flex w-full max-w-md flex-col gap-6">
      <QuotaMeter label="Within budget" used={32_400} />
      <QuotaMeter label="Approaching limit" used={132_000} />
      <QuotaMeter label="Limit reached" used={150_000} />
    </div>
  );
}
