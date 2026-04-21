import { Button, DateTime, type Range, type TimeUnit } from "@unkey/ui";
import { useState } from "react";
import { Preview } from "../../../components/Preview";

const INITIAL_RANGE: Range = {
  from: new Date("2026-04-14T00:00:00Z"),
  to: new Date("2026-04-16T23:59:59Z"),
};

export function RangeExample() {
  const [range, setRange] = useState<Range | undefined>(INITIAL_RANGE);
  const [start, setStart] = useState<TimeUnit>({ HH: "00", mm: "00", ss: "00" });
  const [end, setEnd] = useState<TimeUnit>({ HH: "23", mm: "59", ss: "59" });

  return (
    <Preview>
      <DateTime
        initialRange={INITIAL_RANGE}
        onChange={(date, s, e) => {
          setRange(date);
          if (s) setStart(s);
          if (e) setEnd(e);
        }}
      >
        <DateTime.Calendar mode="range" />
        <DateTime.TimeInput type="range" />
        <DateTime.Actions>
          <div className="text-xs text-gray-11">
            {range?.from ? range.from.toDateString() : "no start"}
            {" "}
            {start.HH}:{start.mm}:{start.ss}
            {" → "}
            {range?.to ? range.to.toDateString() : "no end"}
            {" "}
            {end.HH}:{end.mm}:{end.ss}
          </div>
        </DateTime.Actions>
      </DateTime>
    </Preview>
  );
}

export function SingleExample() {
  const [picked, setPicked] = useState<Date | undefined>(
    new Date("2026-04-16T12:00:00Z"),
  );

  return (
    <Preview>
      <DateTime
        initialRange={{ from: new Date("2026-04-16T12:00:00Z"), to: undefined }}
        onChange={(date) => setPicked(date?.from)}
      >
        <DateTime.Calendar mode="single" />
        <DateTime.TimeInput type="single" />
        <DateTime.Actions>
          <div className="text-xs text-gray-11">
            {picked ? picked.toDateString() : "No date selected"}
          </div>
        </DateTime.Actions>
      </DateTime>
    </Preview>
  );
}

export function BoundedExample() {
  const min = new Date("2026-04-10T00:00:00Z");
  const max = new Date("2026-04-25T23:59:59Z");
  const [range, setRange] = useState<Range | undefined>({
    from: new Date("2026-04-16T12:00:00Z"),
    to: undefined,
  });

  return (
    <Preview>
      <DateTime
        minDate={min}
        maxDate={max}
        initialRange={range}
        onChange={(date) => setRange(date)}
      >
        <DateTime.Calendar mode="range" />
        <DateTime.TimeInput type="range" />
        <DateTime.Actions>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setRange({ from: undefined, to: undefined })}
          >
            Clear
          </Button>
        </DateTime.Actions>
      </DateTime>
    </Preview>
  );
}
