import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { Badge, CopyButton, InfoTooltip } from "@unkey/ui";
import { STATUS_STYLES } from "./status-cell";

type BadgeListProps = {
  log: KeyDetailsLog;
  selectedLog: KeyDetailsLog | null;
  maxTags?: number;
};

/**
 * Single status item for log like KEY_VERIFICATION_OUTCOMES
 * @param log for the row
 * @param selectedLog for log selected in table
 * @returns JSX element that shows status outcomes of log
 */

export const BadgeList = ({ log, selectedLog, maxTags = 4 }: BadgeListProps) => {
  return (
    <div className="flex flex-nowrap gap-1 items-center">
      {log.tags && log.tags.length > 0 ? (
        log.tags.slice(0, maxTags).map((tag) => (
          <InfoTooltip
            className="px-2 py-1"
            key={tag}
            content={
              <div className="max-w-xs">
                {tag.length > 60 ? (
                  <div>
                    <div className="break-all max-w-[300px] truncate">{tag}</div>
                    <div className="flex items-center justify-between mt-1.5">
                      <div className="text-xs opacity-60">({tag.length} characters)</div>
                      {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                      <div className="pointer-events-auto" onClick={(e) => e.stopPropagation()}>
                        <CopyButton variant="ghost" value={tag} />
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="flex justify-between items-center gap-1.5">
                    <div className="break-all max-w-[300px] truncate">{tag}</div>
                    {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                    <div
                      className="pointer-events-auto flex-shrink-0"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <CopyButton variant="ghost" value={tag} />
                    </div>
                  </div>
                )}
              </div>
            }
            position={{ side: "top", align: "start", sideOffset: 5 }}
            asChild
          >
            <Badge
              className={cn(
                "whitespace-nowrap max-w-[150px] truncate",
                selectedLog?.request_id === log.request_id
                  ? STATUS_STYLES.success.badge.selected
                  : "",
              )}
            >
              {shortenId(tag, {
                endChars: 0,
                minLength: 14,
                startChars: 10,
              })}
            </Badge>
          </InfoTooltip>
        ))
      ) : (
        <span className="text-gray-8">—</span>
      )}
      {log.tags && log.tags.length > maxTags && (
        <InfoTooltip
          variant="inverted"
          content={
            <div className="flex flex-col gap-2 py-1 max-w-xs max-h-[300px] overflow-y-auto">
              <div className="text-xs opacity-75 font-medium">
                {log.tags.length - maxTags} more tags:
              </div>
              {log.tags.slice(maxTags).map((tag, idx) => (
                <div key={idx + tag} className="text-xs">
                  {tag.length > 60 ? (
                    <div>
                      <div className="break-all max-w-[300px] truncate">{tag}</div>
                      <div className="flex items-center justify-between mt-1.5">
                        <div className="text-xs opacity-60">({tag.length} characters)</div>
                        {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                        <div className="pointer-events-auto" onClick={(e) => e.stopPropagation()}>
                          <CopyButton variant="ghost" value={tag} />
                        </div>
                      </div>
                    </div>
                  ) : (
                    <div className="flex justify-between items-start gap-1.5">
                      <div className="break-all max-w-[300px] truncate">{tag}</div>
                      {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
                      <div
                        className="pointer-events-auto flex-shrink-0"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <CopyButton variant="ghost" value={tag} />
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          }
          position={{ side: "top", align: "start", sideOffset: 5 }}
          asChild
        >
          <Badge
            className={cn(
              "whitespace-nowrap",
              selectedLog?.request_id === log.request_id
                ? STATUS_STYLES.success.badge.selected
                : "",
            )}
          >
            +{log.tags.length - maxTags}
          </Badge>
        </InfoTooltip>
      )}
    </div>
  );
};
