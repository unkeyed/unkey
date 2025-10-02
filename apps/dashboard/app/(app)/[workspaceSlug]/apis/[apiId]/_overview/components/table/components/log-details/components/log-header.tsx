import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { XMark } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";

export const LogHeader = ({
  onClose,
  log,
}: {
  onClose: () => void;
  log: KeysOverviewLog;
}) => {
  return (
    <div className="border-b-[1px] flex justify-between items-center border-gray-4 h-[50px] px-4 py-2">
      <div className="flex gap-2 items-center flex-1 min-w-0">
        <Badge
          className={cn("uppercase px-[6px] rounded-md font-mono", {
            "bg-success-3 text-success-11 hover:bg-success-4": log.key_details?.enabled,
            "bg-error-3 text-error-11 hover:bg-error-4": !log.key_details?.enabled,
          })}
        >
          {log.key_details?.enabled ? "Active" : "Disabled"}
        </Badge>
        <p className="text-xs text-accent-12 truncate flex-1">
          {log.key_details?.name || log.key_id}
        </p>
      </div>
      <div className="flex gap-1 items-center shrink-0">
        <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
          <XMark className="text-grayA-9 stroke-2" size="sm-regular" />
        </Button>
      </div>
    </div>
  );
};
