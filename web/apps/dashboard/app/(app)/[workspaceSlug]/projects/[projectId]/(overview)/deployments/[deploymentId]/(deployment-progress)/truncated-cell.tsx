import { cn } from "@/lib/utils";
import { InfoTooltip } from "@unkey/ui";

type Props = {
  text: string;
  threshold?: number;
  maxWidth?: string;
  className?: string;
  side?: "top" | "bottom";
};

export function TruncatedCell({
  text,
  threshold = 80,
  maxWidth = "max-w-[300px]",
  className,
  side = "bottom",
}: Props) {
  return (
    <InfoTooltip
      content={<div className="whitespace-pre-wrap font-mono text-xs break-all w-125">{text}</div>}
      position={{ side, align: "start" }}
      asChild
      disabled={!text.includes("\n") && text.length < threshold}
    >
      <div className={cn("font-mono text-xs text-pretty my-2", maxWidth, className)}>{text}</div>
    </InfoTooltip>
  );
}
