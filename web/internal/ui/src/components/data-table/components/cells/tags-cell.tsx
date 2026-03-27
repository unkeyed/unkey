import { cn } from "../../../../lib/utils";
import { Badge } from "../../../badge";
import { CopyButton } from "../../../buttons/copy-button";
import { InfoTooltip } from "../../../info-tooltip";
import { STATUS_STYLES } from "../../constants/constants";

export type TagsCellProps = {
  tags: string[];
  isSelected: boolean;
  /** Max visible tags before showing "+N" overflow (default: 3) */
  maxVisible?: number;
  /** Options for shortening tag display text */
  shortenOptions?: {
    startChars?: number;
    endChars?: number;
    separator?: string;
    minLength?: number;
  };
};

function shortenTag(
  tag: string,
  options: { startChars?: number; endChars?: number; separator?: string; minLength?: number } = {},
): string {
  const { startChars = 10, endChars = 0, separator = "...", minLength = 14 } = options;

  if (tag.length <= minLength || startChars + endChars >= tag.length) {
    return tag;
  }

  const [prefix, rest] = tag.includes("_") ? tag.split("_", 2) : [null, tag];
  let s = "";
  if (prefix) {
    s += prefix;
    s += "_";
  }
  s += rest.substring(0, startChars);
  s += separator;
  if (endChars > 0) {
    s += rest.substring(rest.length - endChars);
  }
  return s;
}

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

export const TagsCell = ({ tags, isSelected, maxVisible = 3, shortenOptions }: TagsCellProps) => {
  if (!tags || tags.length === 0) {
    return <span className="text-gray-8">—</span>;
  }

  return (
    <div className="flex flex-wrap gap-1 items-center">
      {tags.slice(0, maxVisible).map((tag) => (
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
              isSelected ? STATUS_STYLES.badge?.selected : "",
            )}
          >
            {shortenTag(tag, shortenOptions)}
          </Badge>
        </InfoTooltip>
      ))}
      {tags.length > maxVisible && (
        <InfoTooltip
          variant="inverted"
          content={
            <div className="flex flex-col gap-2 py-1 max-w-xs max-h-[300px] overflow-y-auto">
              <div className="text-xs opacity-75 font-medium">
                {tags.length - maxVisible} more tags:
              </div>
              {tags.slice(maxVisible).map((tag, idx) => (
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
              isSelected ? STATUS_STYLES.badge?.selected : "",
            )}
          >
            +{tags.length - maxVisible}
          </Badge>
        </InfoTooltip>
      )}
    </div>
  );
};
