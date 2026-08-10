import { Meter, MeterHeader, MeterIndicator, MeterLabel, MeterTrack, MeterValue } from "@unkey/ui";

export default function BasicMeter() {
  return (
    <Meter value={32_400} max={150_000}>
      <MeterHeader>
        <MeterLabel>Key verifications this month</MeterLabel>
        <MeterValue>{() => "32,400 / 150,000"}</MeterValue>
      </MeterHeader>
      <MeterTrack>
        <MeterIndicator className="bg-info-9" />
      </MeterTrack>
    </Meter>
  );
}
