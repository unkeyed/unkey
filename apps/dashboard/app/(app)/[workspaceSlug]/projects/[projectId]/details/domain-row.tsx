import { CircleCheck, Link4, ShareUpRight } from "@unkey/icons";
import { Badge } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { Card } from "./card";

type DomainRowProps = {
  domain: string;
  className?: string;
};

export function DomainRow({ domain, className }: DomainRowProps) {
  return (
    <div
      className={cn(
        "border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] last:border-b w-full px-4 py-3 flex justify-between items-center",
        className
      )}
    >
      <div className="flex items-center">
        <Link4 className="text-gray-9" size="sm-medium" />
        <Link
          href={`https://${domain}`}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center ml-3 transition-all hover:underline decoration-dashed underline-offset-2"
        >
          <div className="text-gray-12 font-medium text-xs mr-2">{domain}</div>
          <ShareUpRight className="text-gray-9 shrink-0" size="md-regular" />
        </Link>
        <div className="ml-3" />
      </div>
      <Badge
        variant="success"
        className="p-[5px] size-[22px] flex items-center justify-center"
      >
        <CircleCheck className="shrink-0" size="sm-regular" />
      </Badge>
    </div>
  );
}

export const DomainRowSkeleton = () => {
  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] last:border-b w-full px-4 py-3 flex justify-between items-center">
      <div className="flex items-center">
        <Link4 className="text-grayA-6" size="sm-medium" />
        <div className="h-3 w-32 bg-grayA-3 rounded animate-pulse ml-3 mr-2" />
        <ShareUpRight className="text-grayA-6 shrink-0" size="md-regular" />
        <div className="ml-3" />
      </div>
      <div className="p-[5px] size-[22px] bg-grayA-3 rounded animate-pulse flex items-center justify-center">
        <div className="size-3 bg-grayA-4 rounded-full" />
      </div>
    </div>
  );
};

export const DomainRowEmpty = () => (
  <Card
    className={
      "rounded-[14px] flex justify-center items-center overflow-hidden border-gray-4 border-dashed bg-gray-1/50 min-h-[150px] relative group hover:border-gray-5 transition-colors duration-200"
    }
  >
    <div className="flex flex-col items-center gap-3 px-6 py-8 text-center">
      {/* Icon with subtle animation */}
      <div className="relative">
        <div className="absolute inset-0 bg-gradient-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 group-hover:opacity-30 transition-opacity duration-300 animate-pulse" />
        <div className="relative bg-gray-3 rounded-full p-3 group-hover:bg-gray-4 transition-all duration-200">
          <Link4
            className="text-gray-9 size-6 group-hover:text-gray-11 transition-all duration-200 animate-pulse"
            style={{ animationDuration: "2s" }}
          />
        </div>
      </div>
      {/* Content */}
      <div className="space-y-2">
        <h3 className="text-gray-12 font-medium text-sm">No domains found</h3>
        <p className="text-gray-9 text-xs max-w-[280px] leading-relaxed">
          Your configured domains will appear here once they're set up and
          verified.
        </p>
      </div>
    </div>
  </Card>
);
