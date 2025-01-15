"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Button, DateTime, type Range, type TimeUnit } from "@unkey/ui";
import { useState } from "react";
export const DateTimeExample: React.FC = () => {
  const today = new Date();
  const [date, setDate] = useState<Range>();
  const [startTime, setStartTime] = useState<TimeUnit>();
  const [endTime, setEndTime] = useState<TimeUnit>();
  const [showApply, setShowApply] = useState(false);

  const handleChange = (
    date: Range | undefined,
    start: TimeUnit | undefined,
    end: TimeUnit | undefined,
  ) => {
    setShowApply(false);
    setDate(date);
    setStartTime(start);
    setEndTime(end);
  };

  const handleApply = () => {
    setShowApply(true);
  };

  const handleReset = () => {
    setDate(undefined);
    setStartTime(undefined);
    setEndTime(undefined);
    setShowApply(false);
  };
  return (
    <RenderComponentWithSnippet>
      <Row>
        <div className="flex flex-col w-full">
          <div className="flex w-full ">
            <div className="w-full border border-1-gray-12">
              On Blur or Selection:
              <p className="m-0 p-0">
                Date Range: <span>{date?.from?.toLocaleDateString() ?? "no date"}</span> -{" "}
                <span>{date?.to?.toLocaleDateString() ?? "no date"}</span>
              </p>
              <p className="m-0 p-0">
                Time Span:{" "}
                <span>
                  {startTime ? `${startTime.HH}:${startTime.mm}:${startTime.ss}` : "no time"}
                </span>{" "}
                - <span>{endTime ? `${endTime.HH}:${endTime.mm}:${endTime.ss}` : "no time"}</span>
              </p>
            </div>
            <div className="w-full border border-1-gray-12">
              {showApply ? (
                <div className="w-1/2 border border-1-gray-12">
                  On Submit button:
                  <p className="m-0 p-0">
                    Date Range: <span>{date?.from?.toLocaleDateString() ?? "no date"}</span> -{" "}
                    <span>{date?.to?.toLocaleDateString() ?? "no date"}</span>
                  </p>
                  <p className="m-0 p-0">
                    Time Span:{" "}
                    <span>
                      {startTime ? `${startTime.HH}:${startTime.mm}:${startTime.ss}` : "no time"}
                    </span>{" "}
                    -{" "}
                    <span>{endTime ? `${endTime.HH}:${endTime.mm}:${endTime.ss}` : "no time"}</span>
                  </p>
                </div>
              ) : null}
            </div>
          </div>
          <div className="flex flex-col w-full pt-12">
            <DateTime onChange={handleChange} maxDate={today}>
              <DateTime.Calendar mode="range" />
              <div className="flex flex-row gap-4 justify-center p-0 m-0">
                <DateTime.TimeInput type="start" />
                <DateTime.TimeInput type="end" />
              </div>
              <DateTime.Actions>
                <Button onClick={handleApply}>Submit</Button>
                <Button onClick={handleReset}>Reset</Button>
              </DateTime.Actions>
            </DateTime>
          </div>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};
