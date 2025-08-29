import { CircleCheck, CircleWarning, Link4, ShareUpRight } from "@unkey/icons";
import { Badge } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type DomainRowProps = {
  domain: string;
  status: "success" | "error";
  tags: string[];
};

export function DomainRow({ domain, status, tags }: DomainRowProps) {
  const statusConfig = {
    success: { variant: "success" as const, icon: CircleCheck },
    error: { variant: "error" as const, icon: CircleWarning },
  };

  const { variant, icon: StatusIcon } = statusConfig[status];

  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] last:border-b w-full px-4 py-3 flex justify-between items-center">
      <div className="flex items-center">
        <Link4 className="text-gray-9" size="sm-medium" />
        <div className="text-gray-12 font-medium text-xs ml-3 mr-2">{domain}</div>
        <ShareUpRight className="text-gray-9 shrink-0" size="md-regular" />
        <div className="ml-3" />
        <div className="flex gap-1.5 items-center h-4">
          {tags.map((tag) => (
            <InfoTag
              key={tag}
              label={tag}
              className={
                tag === "https" ? "bg-gray-4 text-grayA-11" : "bg-feature-4 text-feature-11"
              }
            />
          ))}
        </div>
      </div>
      <Badge variant={variant} className="p-[5px] size-[22px] flex items-center justify-center">
        <StatusIcon className="shrink-0" size="sm-regular" />
      </Badge>
    </div>
  );
}

function InfoTag({ label, className }: { label: string; className?: string }) {
  return (
    <div className={cn("rounded-md px-1 text-[11px] leading-6 font-mono", className)}>{label}</div>
  );
}
