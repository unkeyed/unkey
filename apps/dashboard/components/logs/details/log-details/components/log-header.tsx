import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import type { StandardLogTypes } from "..";

type Props = {
  log: StandardLogTypes;
  onClose: () => void;
};

export const LogHeader = ({ onClose, log }: Props) => {
  return (
    <div className="border-b-[1px] flex justify-between items-center border-gray-4 h-[50px] px-4 py-2">
      <div className="flex gap-2 items-center min-w-0">
        <Badge className="uppercase px-[6px] rounded-md font-mono bg-accent-3 text-accent-11 hover:bg-accent-4">
          {log.method}
        </Badge>
        <p className="text-xs text-accent-12 truncate flex-1">{log.path} </p>

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
      </div>

      <div className="flex gap-1 items-center shrink-0">
        <div className="flex gap-3">
          <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
            <XMark className="text-grayA-9 stroke-2" size="sm-regular" />
          </Button>
        </div>
      </div>
    </div>
  );
};
