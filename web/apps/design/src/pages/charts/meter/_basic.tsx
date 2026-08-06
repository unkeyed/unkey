import { Meter, MeterHeader, MeterLabel, MeterTrack, MeterValue } from "@unkey/ui";

export function BasicMeter() {
  return (
    <div className="w-full max-w-md">
      <Meter>
        <MeterHeader>
          <MeterLabel>Key verifications this month</MeterLabel>
          <MeterValue>32,400 / 150,000</MeterValue>
        </MeterHeader>
        <MeterTrack fraction={0.216} fillClassName="bg-info-9" />
      </Meter>
    </div>
  );
}
