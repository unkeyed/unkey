import { Meter, MeterIndicator, MeterLabel, MeterTrack, MeterValue } from "@unkey/ui";

export default function InlineMeter() {
  return (
    <Meter layout="inline" value={7_700_000} max={10_000_000}>
      <MeterLabel className="w-40 shrink-0">Monthly API operations</MeterLabel>
      <MeterValue className="w-24 shrink-0 text-right font-normal text-gray-11">
        {() => "7,700,000"}
      </MeterValue>
      <MeterTrack>
        <MeterIndicator />
      </MeterTrack>
      <span className="w-24 shrink-0 text-right text-[13px] text-gray-12 tabular-nums">
        10,000,000
      </span>
    </Meter>
  );
}
