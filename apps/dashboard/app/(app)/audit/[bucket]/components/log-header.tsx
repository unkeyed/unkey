import { Badge } from "@/components/ui/badge";
import { Button } from "@unkey/ui";
import { X } from "lucide-react";
import { Data } from "./table/types";

type Props = {
  log: Data;
  onClose: () => void;
};

export const LogHeader = ({ onClose, log }: Props) => {
  console.log(log);
  return (
    <div className="border-b-[1px] px-3 py-4 flex justify-between border-border items-center">
      <div className="flex gap-2 items-center overflow-hidden">
        <Badge variant="secondary" className="bg-transparent shrink-0">
          {log.auditLog.event}
        </Badge>
      </div>
      <div className="flex gap-1 items-center shrink-0">
        <Button shape="square" variant="ghost" onClick={onClose}>
          <X
            size="22"
            strokeWidth="1.5"
            className="text-content/65 cursor-pointer"
          />
        </Button>
      </div>
    </div>
  );
};
