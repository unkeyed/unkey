import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import type { Log } from "../../data";

type Props = {
  log: Log;
  onClose: () => void;
};
export const LogHeader = ({ onClose, log }: Props) => {
  return (
    <div className="border-b-[1px] px-3 py-4 flex justify-between border-border items-center">
      <div className="flex gap-2">
        <Badge variant="secondary" className="bg-transparent">
          POST
        </Badge>
        <p className="text-[13px] text-content/65">{log.path}</p>
      </div>

      <div className="flex gap-1 items-center">
        <Badge
          className={cn(
            "bg-background border border-solid border-border text-current hover:bg-transparent",
            log.response_status >= 400 && "border-red-6 text-red-11"
          )}
        >
          {log.response_status}
        </Badge>

        <span className="text-content/65">|</span>
        <X
          onClick={onClose}
          size="22"
          strokeWidth="1.5"
          className="text-content/65 cursor-pointer"
        />
      </div>
    </div>
  );
};
