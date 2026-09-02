import { IconKeyOutline18, IconMathFunctionOutline18 } from "nucleo-ui-outline-18";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";

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
            {`${user.firstName ?? ""} ${user.lastName ?? ""}`.trim() ||
              user.username ||
              log.auditLog.actor.id}
          </span>
        ) : (
          <>
            {isKey ? (
              <IconKeyOutline18 className="size-3" />
            ) : (
              <IconMathFunctionOutline18 className="size-3" />
            )}
            <span className="font-mono text-xs truncate secret">{log.auditLog.actor.id}</span>
          </>
        )}
      </div>
    </div>
  );
};
