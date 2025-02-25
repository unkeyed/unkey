import { Badge } from "@/components/ui/badge";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";

type Props = {
  log: AuditLog;
  onClose: () => void;
};

export const LogHeader = ({ onClose, log }: Props) => {
  return (
    <div className="border-b-[1px] flex justify-between items-center border-gray-4 pb-3 w-full">
      <div className="flex gap-2 items-center flex-1 min-w-0">
        <Badge className="uppercase px-[6px] rounded-md font-mono bg-accent-3 text-accent-11 hover:bg-accent-4">
          {log.auditLog.event}
        </Badge>
      </div>

      <div className="flex gap-1 items-center ">
        <div className="flex gap-3">
          <Button size="icon" variant="ghost" onClick={onClose} className="[&_svg]:size-3">
            <XMark className="text-gray-12 stroke-2" />
          </Button>
        </div>
      </div>
    </div>
  );
};
