import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { Log } from "@unkey/clickhouse/src/logs";
import { XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";

type Props = {
  log: Log;
  onClose: () => void;
};

export const LogHeader = ({ onClose, log }: Props) => {
  return (
    <div className="border-b-[1px] flex justify-between items-center border-gray-4 pb-3">
      <div className="flex gap-2 items-center flex-1 min-w-0">
        <Badge className="uppercase px-[6px] rounded-md font-mono bg-accent-3 text-accent-11 hover:bg-accent-4">
          {log.method}
        </Badge>
        <p className="text-xs text-accent-12 truncate flex-1 mr-4">{log.path} </p>
      </div>

      <div className="flex gap-1 items-center shrink-0">
        <div className="flex gap-3">
          <Badge
            className={cn("px-[6px] rounded-md font-mono text-xs", {
              "bg-success-3 text-success-11 hover:bg-success-4":
                log.response_status >= 200 && log.response_status < 300,
              "bg-warning-3 text-warning-11 hover:bg-warning-4":
                log.response_status >= 400 && log.response_status < 500,
              "bg-error-3 text-error-11 hover:bg-error-4": log.response_status >= 500,
            })}
          >
            {log.response_status}
          </Badge>
          <span className="text-gray-5">|</span>
          <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
            <XMark className="text-gray-12 stroke-2" />
          </Button>
        </div>
      </div>
    </div>
  );
};
