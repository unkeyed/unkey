import { IconCheckOutline12 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import type { ReactNode } from "react";
import type { StepStatus } from "../../model/steps";

export function StepCard({
  number,
  title,
  label,
  status,
  summary,
  onOpen,
  children,
}: {
  number: number;
  title: string;
  label: string;
  status: StepStatus;
  summary?: ReactNode;
  onOpen?: () => void;
  children?: ReactNode;
}) {
  if (status === "done") {
    return (
      <div className="flex min-h-12 items-center gap-3 rounded-lg border border-grayA-4 bg-raised px-4 py-2">
        <span className="flex size-[18px] shrink-0 items-center justify-center rounded-full border border-grayA-6 text-gray-11">
          <IconCheckOutline12 className="size-2.5" />
        </span>
        <span className="w-20 shrink-0 text-sm font-medium text-gray-9">{label}</span>
        <div className="flex min-w-0 flex-1 items-center gap-2 text-sm text-gray-12">{summary}</div>
        {onOpen ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            aria-label={`Edit ${label.toLowerCase()}`}
            onClick={onOpen}
          >
            Edit
          </Button>
        ) : null}
      </div>
    );
  }

  if (status === "todo") {
    return (
      <div className="flex items-center gap-3 rounded-lg border border-grayA-4 px-4 py-3">
        <StepNumber number={number} muted />
        <span className="text-sm font-medium text-gray-9">{title}</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 rounded-lg border border-grayA-4 bg-raised p-4">
      <div className="flex items-center gap-3">
        <StepNumber number={number} />
        <span className="text-sm font-medium text-gray-12">{title}</span>
      </div>
      {children}
    </div>
  );
}

function StepNumber({ number, muted = false }: { number: number; muted?: boolean }) {
  return (
    <span
      className={cn(
        "flex size-[18px] shrink-0 items-center justify-center rounded-full text-2xs font-semibold",
        muted ? "bg-grayA-5 text-gray-11" : "bg-gray-12 text-gray-1",
      )}
    >
      {number}
    </span>
  );
}
