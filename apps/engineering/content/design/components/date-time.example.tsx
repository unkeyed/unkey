"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button, DateTime, type Range } from "@unkey/ui";
import { useState } from "react";

type TimeUnit = {
  HH: string;
  mm: string;
  ss: string;
};

export const DateTimeExample: React.FC = () => {
  const [date, setDate] = useState<Range>();
  const [startTime, setStartTime] = useState<TimeUnit>();
  const [endTime, setEndTime] = useState<TimeUnit>();
  const suggestions = [
    { label: "Last 5 minutes", relativeTime: 5 * 60 * 1000 },
    { label: "Last 15 minutes", relativeTime: 15 * 60 * 1000 },
    { label: "Last 30 minutes", relativeTime: 30 * 60 * 1000 },
    { label: "Last 1 hour", relativeTime: 60 * 60 * 1000 },
    { label: "Last 3 hours", relativeTime: 3 * 60 * 60 * 1000 },
    { label: "Last 6 hours", relativeTime: 6 * 60 * 60 * 1000 },
    { label: "Last 12 hours", relativeTime: 12 * 60 * 60 * 1000 },
    { label: "Last 24 hours", relativeTime: 24 * 60 * 60 * 1000 },
    { label: "Last 2 days", relativeTime: 2 * 24 * 60 * 60 * 1000 },
  ];
  const handleApply = (newDate?: Range, newStartTime?: TimeUnit, newEndTime?: TimeUnit) => {
    newDate ? setDate(newDate) : null;
    newStartTime ? setStartTime(newStartTime) : null;
    newEndTime ? setEndTime(newEndTime) : null;
  };
  const handleChange = (newDate?: Range, newStartTime?: TimeUnit, newEndTime?: TimeUnit) => {
    newDate ? setDate(newDate) : null;
    newStartTime ? setStartTime(newStartTime) : null;
    newEndTime ? setEndTime(newEndTime) : null;
  };

  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="flex flex-col w-full">
          <div className="w-full p-4 border border-1-gray-12 h-32">
            <div className="w-full ">
              <p className="m-0 p-0">
                Date Range: <span>{date?.from?.toLocaleDateString() ?? "no date"}</span> -{" "}
                <span>{date?.to?.toLocaleDateString() ?? "no date"}</span>
              </p>
              <p className="m-0 p-0">
                Time Span: <span>{JSON.stringify(startTime)}</span> -{" "}
                <span>{JSON.stringify(endTime)}</span>
              </p>
            </div>
          </div>
          <div className="flex w-full pt-12 justify-center items-center">
            <DateTime
              onChange={(date?: Range, startTime?: TimeUnit, endTime?: TimeUnit) =>
                handleChange(date, startTime, endTime)
              }
            >
              <DateTime.Suggestions suggestions={suggestions} />
              <div className="flex flex-col">
                <DateTime.Calendar mode="range" />
                <DateTime.TimeInput type="range" />
                <DateTime.Actions>
                  <Button className="w-full" onClick={() => handleApply} variant={"primary"}>
                    Apply Filter
                  </Button>
                </DateTime.Actions>
              </div>
            </DateTime>
          </div>
          <ul>
            <h4>List left to do:</h4>
            <li>AM/PM in time component</li>
            <li>Timezone text</li>
            <li>Combine TimeInput components into a single row</li>
            <li>Use same Mode as calendar "single" or "range"</li>
            <li>Handle style inside component</li>
            <li>Handle keyboard navigation inside time component</li>
            <li>enter 2 digit auto move to next input</li>
            <li>If single digit and : is used next input</li>
          </ul>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};
