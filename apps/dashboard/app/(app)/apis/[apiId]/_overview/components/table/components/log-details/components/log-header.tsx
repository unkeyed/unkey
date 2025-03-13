import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const LogHeader = ({
  onClose,
  log,
}: {
  onClose: () => void;
  log: KeysOverviewLog;
}) => {
  return (
    <div className="border-b-[1px] flex justify-between items-center border-gray-4 pb-3">
      <div className="flex gap-2 items-center flex-1 min-w-0">
        <Badge
          className={cn("uppercase px-[6px] rounded-md font-mono", {
            "bg-success-3 text-success-11 hover:bg-success-4": log.key_details?.enabled,
            "bg-error-3 text-error-11 hover:bg-error-4": !log.key_details?.enabled,
          })}
        >
          {log.key_details?.enabled ? "Active" : "Disabled"}
        </Badge>
        <p className="text-xs text-accent-12 truncate flex-1 mr-4">
          {log.key_details?.name || log.key_id}{" "}
        </p>
      </div>
      <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
        <XMark className="text-gray-12 stroke-2" />
      </Button>
    </div>
  );
};
