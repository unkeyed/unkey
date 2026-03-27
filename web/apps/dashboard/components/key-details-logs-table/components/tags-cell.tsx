import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { Badge, CopyButton, InfoTooltip } from "@unkey/ui";
import { STATUS_STYLES } from "../utils/get-row-class";

type TagsCellProps = {
  log: KeyDetailsLog;
  isSelected: boolean;
};

const TagTooltipContent = ({ tag }: { tag: string }) => {
  if (tag.length > 60) {
    return (
      <div>
        <div className="break-all max-w-[300px] truncate">{tag}</div>
        <div className="flex items-center justify-between mt-1.5">
          <div className="text-xs opacity-60">({tag.length} characters)</div>
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: click handler for copy */}
          <div className="pointer-events-auto" onClick={(e) => e.stopPropagation()}>
            <CopyButton variant="ghost" value={tag} />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex justify-between items-center gap-1.5">
      <div className="break-all max-w-[300px] truncate">{tag}</div>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: click handler for copy */}
      <div className="pointer-events-auto shrink-0" onClick={(e) => e.stopPropagation()}>
        <CopyButton variant="ghost" value={tag} />
      </div>
    </div>
  );
};

export const TagsCell = ({ log, isSelected }: TagsCellProps) => {
  if (!log.tags || log.tags.length === 0) {
    return <span className="text-gray-8">—</span>;
  }

  return (
    <div className="flex flex-wrap gap-1 items-center">
      {log.tags.slice(0, 3).map((tag) => (
        <InfoTooltip
          variant="inverted"
          className="px-2 py-1"
          key={tag}
          content={
            <div className="max-w-xs">
              <TagTooltipContent tag={tag} />
            </div>
          }
          position={{ side: "top", align: "start", sideOffset: 5 }}
          asChild
        >
          <Badge
            className={cn(
              "whitespace-nowrap max-w-[150px] truncate",
              isSelected ? STATUS_STYLES.success.badge?.selected : "",
            )}
          >
            {shortenId(tag, {
              endChars: 0,
              minLength: 14,
              startChars: 10,
            })}
          </Badge>
        </InfoTooltip>
      ))}
      {log.tags.length > 3 && (
        <InfoTooltip
          variant="inverted"
          content={
            <div className="flex flex-col gap-2 py-1 max-w-xs max-h-[300px] overflow-y-auto">
              <div className="text-xs opacity-75 font-medium">
                {log.tags.length - 3} more tags:
              </div>
              {log.tags.slice(3).map((tag, idx) => (
                <div key={idx + tag} className="text-xs">
                  <TagTooltipContent tag={tag} />
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
              isSelected ? STATUS_STYLES.success.badge?.selected : "",
            )}
          >
            +{log.tags.length - 3}
          </Badge>
        </InfoTooltip>
      )}
    </div>
  );
};
