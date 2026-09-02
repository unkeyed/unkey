import { Badge } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { IconCircleCheckOutline12, IconLink4Outline12 } from "nucleo-ui-outline-12";
import { IconShareUpRightOutline18 } from "nucleo-ui-outline-18";

type DomainRowProps = {
  domain: string;
  className?: string;
};

export function DomainRow({ domain, className }: DomainRowProps) {
  return (
    <div
      className={cn(
        "border border-gray-4 border-t-0 first:border-t first:rounded-t-lg last:rounded-b-lg last:border-b w-full px-4 py-3 flex justify-between items-center",
        className,
      )}
    >
      <div className="flex items-center">
        <IconLink4Outline12 className="text-gray-9" />
        <Link
          href={`https://${domain}`}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center ml-3 transition-all hover:underline decoration-dashed underline-offset-2"
        >
          <div className="text-gray-12 font-medium text-xs mr-2">{domain}</div>
          <IconShareUpRightOutline18 className="size-4 text-gray-9 shrink-0" />
        </Link>
        <div className="ml-3" />
      </div>
      <Badge variant="success" className="p-[5px] size-[22px] flex items-center justify-center">
        <IconCircleCheckOutline12 className="shrink-0" />
      </Badge>
    </div>
  );
}

export const DomainRowSkeleton = () => {
  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-lg last:rounded-b-lg last:border-b w-full px-4 py-3 flex justify-between items-center">
      <div className="flex items-center">
        <IconLink4Outline12 className="text-grayA-6" />
        <div className="h-3 w-32 bg-grayA-3 rounded-sm animate-pulse ml-3 mr-2" />
        <IconShareUpRightOutline18 className="size-4 text-grayA-6 shrink-0" />
        <div className="ml-3" />
      </div>
      <div className="p-[5px] size-[22px] bg-grayA-3 rounded-sm animate-pulse flex items-center justify-center">
        <div className="size-3 bg-grayA-4 rounded-full" />
      </div>
    </div>
  );
};
