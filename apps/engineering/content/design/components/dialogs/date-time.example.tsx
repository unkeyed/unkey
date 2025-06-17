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
                Time Span: <span>{`${startTime?.HH}:${startTime?.mm}:${startTime?.ss}`}</span> -{" "}
                <span>{`${endTime?.HH}:${endTime?.mm}:${endTime?.ss}`}</span>
              </p>
            </div>
          </div>
          <div className="flex flex-col w-80 mx-auto pt-12 justify-center items-center">
            <DateTime
              onChange={(date?: Range, startTime?: TimeUnit, endTime?: TimeUnit) =>
                handleChange(date, startTime, endTime)
              }
            >
              <DateTime.Calendar mode="range" />
              <DateTime.TimeInput type="range" />
              <DateTime.Actions className="px-2">
                <Button
                  className="w-full"
                  onClick={() => handleApply(date, startTime, endTime)}
                  variant={"primary"}
                >
                  Apply Filter
                </Button>
              </DateTime.Actions>
            </DateTime>
          </div>
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};
