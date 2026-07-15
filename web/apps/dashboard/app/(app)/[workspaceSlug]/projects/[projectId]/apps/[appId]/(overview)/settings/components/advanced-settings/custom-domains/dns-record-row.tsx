import { cn } from "@/lib/utils";
import { CircleCheck, Clock } from "@unkey/icons";
import { CopyButton } from "@unkey/ui";

type DnsRecordRowProps = {
  type: string;
  name: string;
  value: string;
  verified: boolean;
  isLast?: boolean;
};

export function DnsRecordRow({ type, name, value, verified, isLast }: DnsRecordRowProps) {
  return (
    <div
      className={cn(
        "grid grid-cols-[64px_1fr_1fr_48px] px-3 py-2 items-center",
        !isLast && "border-b border-gray-3",
        verified && "text-gray-8",
      )}
    >
      <span className={cn("font-medium", verified ? "text-gray-8" : "text-gray-11")}>{type}</span>
      <span className="flex items-center gap-1.5 min-w-0">
        <code className="font-mono truncate max-w-[200px]">{name}</code>
        {!verified && <CopyButton value={name} className="size-5 shrink-0" variant="ghost" />}
      </span>
      <span className="flex items-center gap-1.5 min-w-0">
        <code className="font-mono truncate max-w-[200px]">{value}</code>
        {!verified && <CopyButton value={value} className="size-5 shrink-0" variant="ghost" />}
      </span>
      <span className="flex justify-center">
        {verified ? (
          <CircleCheck className="size-3.5! text-success-9" />
        ) : (
          <Clock className="size-3.5! text-gray-8" />
        )}
      </span>
    </div>
  );
}
