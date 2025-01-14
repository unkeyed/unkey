"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { DateTime, type Range } from "@unkey/ui";
import { useState } from "react";

export const DateTimeExample: React.FC = () => {
  const [date, setDate] = useState<Range>();
  const [startTime, setStartTime] = useState<string>("09:00");
  const [endTime, setEndTime] = useState<string>("17:00");

  const handleChange = (date: Range) => {
    setDate(date);
    console.log("New Date:", date);
    console.log("New Start Time:", startTime);
    console.log("New End Time:", endTime);
  };
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="w-full">
          <p>
            From Date: <span>{date?.from?.toLocaleDateString() ?? "no date"}</span>
          </p>
          <p>
            To Date: <span>{date?.to?.toLocaleDateString() ?? "no date"}</span>
          </p>
          <DateTime onChange={handleChange}>
            <DateTime.Calendar mode="range"/>
            <div className="flex flex-col gap-4">
              {/* <DateTime.TimeInput type="start" />
            <DateTime.TimeInput type="end" /> */}
              {/* <DateTime.Actions onApply={handleApply} /> */}
            </div>
          </DateTime>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};
