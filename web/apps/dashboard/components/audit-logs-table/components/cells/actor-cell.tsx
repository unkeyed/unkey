import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { Key, MathFunction } from "@unkey/icons";

type ActorCellProps = {
  log: AuditLog;
};

export const ActorCell = ({ log }: ActorCellProps) => {
  const { user } = log;
  const isUser = log.auditLog.actor.type === "user" && user;
  const isKey = log.auditLog.actor.type === "key";

  return (
    <div className="flex items-center gap-3 truncate">
      <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
        {isUser ? (
          <span className="text-xs whitespace-nowrap secret truncate">
            {`${user.firstName ?? ""} ${user.lastName ?? ""}`}
          </span>
        ) : (
          <>
            {isKey ? <Key iconSize="sm-thin" /> : <MathFunction iconSize="sm-thin" />}
            <span className="font-mono text-xs truncate secret">{log.auditLog.actor.id}</span>
          </>
        )}
      </div>
    </div>
  );
};
