import { Meter, MeterHeader, MeterIndicator, MeterLabel, MeterTrack, MeterValue } from "@unkey/ui";

export function BasicMeter() {
  return (
    <div className="w-full max-w-md">
      <Meter value={32_400} max={150_000}>
        <MeterHeader>
          <MeterLabel>Key verifications this month</MeterLabel>
          <MeterValue>{() => "32,400 / 150,000"}</MeterValue>
        </MeterHeader>
        <MeterTrack>
          <MeterIndicator className="bg-info-9" />
        </MeterTrack>
      </Meter>
    </div>
  );
}
