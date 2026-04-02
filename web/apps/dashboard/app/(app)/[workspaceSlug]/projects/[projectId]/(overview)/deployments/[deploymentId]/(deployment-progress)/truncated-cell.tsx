import { cn } from "@/lib/utils";
import { InfoTooltip } from "@unkey/ui";

type Props = {
  text: string;
  className?: string;
  side?: "top" | "bottom";
};

export function TruncatedCell({ text, className, side = "bottom" }: Props) {
  return (
    <InfoTooltip
      content={
        <div className="whitespace-pre-wrap font-mono text-xs break-all text-pretty max-w-125">
          {text}
        </div>
      }
      position={{ side, align: "start" }}
      asChild
    >
      <div
        className={cn(
          "whitespace-pre-wrap font-mono text-xs break-all text-pretty max-w-125 my-2",
          className,
        )}
      >
        {text}
      </div>
    </InfoTooltip>
  );
}
