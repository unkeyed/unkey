import { TimestampInfo, TooltipProvider } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

// A stable anchor so server-render and client-render agree on the tooltip
// text. Real usage would pass Date.now() or a server-supplied timestamp.
const NOW = new Date("2026-04-16T12:00:00.000Z").getTime();
const ONE_MINUTE = 60_000;
const ONE_HOUR = 60 * ONE_MINUTE;
const ONE_DAY = 24 * ONE_HOUR;

export function BasicExample() {
  return (
    <Preview>
      <TooltipProvider>
        <TimestampInfo value={NOW - 5 * ONE_MINUTE} />
      </TooltipProvider>
    </Preview>
  );
}

export function DisplayTypesExample() {
  return (
    <Preview>
      <TooltipProvider>
        <div className="flex flex-col gap-3 items-start font-mono">
          <TimestampInfo value={NOW - 2 * ONE_HOUR} displayType="local" />
          <TimestampInfo value={NOW - 2 * ONE_HOUR} displayType="utc" />
          <TimestampInfo value={NOW - 2 * ONE_HOUR} displayType="relative" />
          <TimestampInfo
            value={NOW - 2 * ONE_HOUR}
            displayType="local_hours_with_millis"
          />
        </div>
      </TooltipProvider>
    </Preview>
  );
}

export function InputFormatsExample() {
  // The component accepts an ISO string, a millisecond epoch, or a 16-digit
  // unix-micro value. Each row below is the same moment in a different form.
  const iso = new Date(NOW - ONE_DAY).toISOString();
  const millis = NOW - ONE_DAY;
  const micros = String((NOW - ONE_DAY) * 1000);
  return (
    <Preview>
      <TooltipProvider>
        <div className="flex flex-col gap-3 items-start">
          <TimestampInfo value={iso} displayType="relative" />
          <TimestampInfo value={millis} displayType="relative" />
          <TimestampInfo value={micros} displayType="relative" />
        </div>
      </TooltipProvider>
    </Preview>
  );
}

export function PositioningExample() {
  return (
    <Preview>
      <TooltipProvider>
        <TimestampInfo
          value={NOW - 30 * ONE_MINUTE}
          displayType="relative"
          side="top"
          align="center"
        />
      </TooltipProvider>
    </Preview>
  );
}
