"use client";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { Badge, CopyButton, InfoTooltip } from "@unkey/ui";
import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { STATUS_STYLES } from "./status-cell";

const BADGE_WIDTH = 158; // 150px max width + 4px gap on each side
const MORE_BUTTON_WIDTH = 60; // Width for "+X" badge

type BadgeListProps = {
  log: KeyDetailsLog;
  selectedLog: KeyDetailsLog | null;
  maxTags?: number;
};

/**
 * Renders a list of tag badges for a log entry with responsive overflow handling
 * @param log for the row
 * @param selectedLog for log selected in table
 * @param Optional setting for to set max items to show in cell
 * @returns JSX element that shows tag badges for the log
 */

export const BadgeList = ({ log, selectedLog, maxTags = 20 }: BadgeListProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(200);

  useLayoutEffect(() => {
    if (!containerRef.current) {
      return;
    }

    // Set initial width before render
    setContainerWidth(containerRef.current.offsetWidth);

    // Observe future changes
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width);
      }
    });

    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, []);

  // Calculate how many badges can fit based on container width
  const visibleTagCount = useMemo(() => {
    if (containerWidth === 0 || !log.tags || log.tags.length === 0) {
      // If no tags or no container width return 0
      return 0;
    }

    // Approximate badge width (max-w-[150px] + padding + gap)

    const availableWidth = containerWidth;

    // First check if all tags (up to maxTags) can fit without reserving space for "+X" badge
    const possibleCount = Math.min(log.tags.length, maxTags);
    const maxBadgesWithoutMoreButton = Math.floor(availableWidth / BADGE_WIDTH);

    if (maxBadgesWithoutMoreButton >= possibleCount) {
      // All tags fit without needing the "+X" badge
      return possibleCount;
    }

    // Tags don't all fit, so reserve space for "+X" badge and calculate how many can be shown
    const maxVisibleBadges = Math.floor((availableWidth - MORE_BUTTON_WIDTH) / BADGE_WIDTH);

    // Show as many as we can fit (at least 1)
    return Math.max(1, maxVisibleBadges);
  }, [containerWidth, log.tags, maxTags]);

  return (
    <div ref={containerRef} className="flex flex-nowrap w-full gap-1 items-center">
      {log.tags && log.tags.length > 0 ? (
        log.tags.slice(0, visibleTagCount).map((tag) => (
          <InfoTooltip
            className="px-2 py-1"
            key={tag}
            content={
              <div className="max-w-xs">
                {tag.length > 60 ? (
                  <div>
                    <div className="max-w-[300px] truncate">{tag}</div>
                    <div className="flex items-center justify-between mt-1.5">
                      <div className="text-xs opacity-60">({tag.length} characters)</div>
                      {/* biome-ignore lint/a11y/useKeyWithClickEvents: CopyButton handles keyboard events */}
                      <div className="pointer-events-auto" onClick={(e) => e.stopPropagation()}>
                        <CopyButton variant="ghost" value={tag} />
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="flex justify-between items-center gap-1.5">
                    <div className="break-all max-w-[300px] truncate">{tag}</div>
                    {/* biome-ignore lint/a11y/useKeyWithClickEvents: CopyButton handles keyboard events */}
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
                minLength: 20,
                startChars: 16,
              })}
            </Badge>
          </InfoTooltip>
        ))
      ) : (
        <span className="text-gray-8">—</span>
      )}
      {log.tags && log.tags.length > visibleTagCount && (
        <InfoTooltip
          content={
            <div className="flex flex-col gap-2 py-1 max-w-xs max-h-[300px] overflow-y-auto">
              <div className="text-xs opacity-75 font-medium">
                {log.tags.length - visibleTagCount} more tags:
              </div>
              {log.tags.slice(visibleTagCount).map((tag, idx) => (
                <div key={idx + tag} className="text-xs">
                  {tag.length > 60 ? (
                    <div>
                      <div className="max-w-[300px] truncate">{tag}</div>
                      <div className="flex items-center justify-between mt-1.5">
                        <div className="text-xs opacity-60">({tag.length} characters)</div>
                        {/* biome-ignore lint/a11y/useKeyWithClickEvents: CopyButton handles keyboard events internally */}
                        <div className="pointer-events-auto" onClick={(e) => e.stopPropagation()}>
                          <CopyButton variant="ghost" value={tag} />
                        </div>
                      </div>
                    </div>
                  ) : (
                    <div className="flex justify-between items-center gap-1.5">
                      <div className="break-all max-w-[300px] truncate">{tag}</div>
                      {/* biome-ignore lint/a11y/useKeyWithClickEvents: CopyButton handles keyboard events internally */}
                      <div
                        className="pointer-events-auto flex-shrink-0 "
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
            +{log.tags.length - visibleTagCount}
          </Badge>
        </InfoTooltip>
      )}
    </div>
  );
};
