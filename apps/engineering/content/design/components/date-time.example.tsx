"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { DateTime, type Range, type TimeUnit } from "@unkey/ui";
import { useState } from "react";
export const DateTimeExample: React.FC = () => {
  const today = new Date();
  const [date, setDate] = useState<Range>();
  const [startTime, setStartTime] = useState<TimeUnit>();
  const [endTime, setEndTime] = useState<TimeUnit>();

  const handleChange = (date: Range) => {
    setDate(date);
    console.log("New Date:", date);
    console.log("New Start Time:", startTime);
    console.log("New End Time:", endTime);
  };
  const handleStartChange = (time: TimeUnit) => {
    setStartTime(time);
  };
  const handleEndChange = (time: TimeUnit) => {
    setEndTime(time);
  };
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="w-full">
          <p>
            From Date: <span>{date?.from?.toLocaleDateString() ?? "no date"}</span>
          </p>
          <p>
            From Time: <span>{startTime ? `${startTime.HH}:${startTime.mm}:${startTime.ss}`: "no time"}</span>
          </p>
          <p>
            To Date: <span>{date?.to?.toLocaleDateString() ?? "no date"}</span>
          </p>
          <p>
            From Time: <span>{endTime ? `${endTime.HH}:${endTime.mm}:${endTime.ss}` : "no time"}</span>
          </p>
          <DateTime onChange={handleChange} maxDate={today}>
            <DateTime.Calendar mode="range"/>
            <div className="flex flex-row gap-4 justify-center p-0 m-0">
              <DateTime.TimeInput type="start" />
              <DateTime.TimeInput type="end" />
              {/* <DateTime.Actions onApply={handleApply} /> */}
            </div>
          </DateTime>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};

