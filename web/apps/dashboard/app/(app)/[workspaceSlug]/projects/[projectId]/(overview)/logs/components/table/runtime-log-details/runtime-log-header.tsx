import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import type { RuntimeLog } from "../../../types";

type Props = {
  log: RuntimeLog;
  onClose: () => void;
};

export const RuntimeLogHeader = ({ onClose, log }: Props) => {
  return (
    <div className="border-b flex justify-between items-center border-gray-4 h-[45px] px-4 py-2">
      <div className="flex gap-2 items-center min-w-0">
        <Badge
          className={cn("uppercase px-[6px] rounded-md font-mono text-xs", {
            "bg-error-3 text-error-11 hover:bg-error-4":
              log.severity === "ERROR" || log.severity === "FATAL",
            "bg-warning-3 text-warning-11 hover:bg-warning-4": log.severity === "WARN",
            "bg-info-3 text-info-11 hover:bg-info-4":
              log.severity === "INFO" || log.severity === "DEBUG",
          })}
        >
          {log.severity}
        </Badge>
      </div>
      <div className="flex gap-1 items-center shrink-0">
        <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
          <XMark className="text-grayA-9 stroke-2" iconSize="sm-regular" />
        </Button>
      </div>
    </div>
  );
};
