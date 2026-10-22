import { FormField, SegmentedControl, SegmentedControlItem } from "@unkey/ui";
import { useState } from "react";

export default function SegmentedControlExample() {
  const [period, setPeriod] = useState<"day" | "week" | "month">("week");

  return (
    <div className="flex w-80 max-w-full flex-col gap-6">
      <FormField label="Reporting period" description="Choose the time range for your report.">
        {(control) => (
          <SegmentedControl
            id={control.id}
            aria-labelledby={`${control.id}-label`}
            aria-describedby={control.describedBy}
            value={period}
            onValueChange={setPeriod}
          >
            <SegmentedControlItem value="day">Day</SegmentedControlItem>
            <SegmentedControlItem value="week">Week</SegmentedControlItem>
            <SegmentedControlItem value="month">Month</SegmentedControlItem>
          </SegmentedControl>
        )}
      </FormField>
      <SegmentedControl aria-label="Disabled period" defaultValue="week" disabled>
        <SegmentedControlItem value="day">Day</SegmentedControlItem>
        <SegmentedControlItem value="week">Week</SegmentedControlItem>
        <SegmentedControlItem value="month">Month</SegmentedControlItem>
      </SegmentedControl>
    </div>
  );
}
