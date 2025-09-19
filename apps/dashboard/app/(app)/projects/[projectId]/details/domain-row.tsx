import { CircleCheck, Link4, ShareUpRight } from "@unkey/icons";
import { Badge } from "@unkey/ui";

type DomainRowProps = {
  domain: string;
};

export function DomainRow({ domain }: DomainRowProps) {
  return (
    <div className="border border-gray-4 border-t-0 first:border-t first:rounded-t-[14px] last:rounded-b-[14px] last:border-b w-full px-4 py-3 flex justify-between items-center">
      <div className="flex items-center">
        <Link4 className="text-gray-9" iconsize="sm-medium" />
        <div className="text-gray-12 font-medium text-xs ml-3 mr-2">
          {domain}
        </div>
        <ShareUpRight className="text-gray-9 shrink-0" iconsize="md-medium" />
        <div className="ml-3" />
      </div>
      <Badge
        variant="success"
        className="p-[5px] size-[22px] flex items-center justify-center"
      >
        <CircleCheck className="shrink-0" iconsize="sm-regular" />
      </Badge>
    </div>
  );
}
