import { SPRING_POP, useLayoutMotion } from "@/hooks/use-layout-motion";
import { IconXmarkOutline12 } from "@unkey/icons";
import { cn } from "cn";
import { useRef } from "react";

interface BaseItem {
  id: string;
  name?: string;
}

interface SelectedItemsListProps<T extends BaseItem> {
  items: T[];
  disabled?: boolean;
  onRemoveItem: (id: string) => void;
  isItemRemovable?: (item: T) => boolean;
  renderIcon: (item: T) => React.ReactNode;
  renderPrimaryText: (item: T) => string;
  renderSecondaryText: (item: T) => string;
  renderBadge?: (item: T) => React.ReactNode;
  className?: string;
  gridCols?: 1 | 2 | 3 | 4;
  itemHeight?: string;
  enableTransitions?: boolean;
}

const ITEM_HIDDEN = { opacity: 0, scale: "0.8", translate: "0 -10px" };
const ITEM_SHOWN = { opacity: 1, scale: "1", translate: "0 0" };
const ITEM_TIMING = { duration: 400, easing: SPRING_POP };

const getGridColsClass = (cols: number) => {
  const gridMap = {
    1: "grid-cols-1",
    2: "grid-cols-2",
    3: "grid-cols-3",
    4: "grid-cols-4",
  } as const;
  return gridMap[cols as keyof typeof gridMap] || "grid-cols-2";
};

export function SelectedItemsList<T extends BaseItem>({
  items,
  disabled = false,
  onRemoveItem,
  isItemRemovable,
  renderIcon,
  renderPrimaryText,
  renderSecondaryText,
  renderBadge,
  className,
  gridCols = 2,
  itemHeight,
  enableTransitions = true,
}: SelectedItemsListProps<T>) {
  const gridRef = useRef<HTMLDivElement>(null);
  useLayoutMotion(gridRef, {
    enabled: enableTransitions,
    move: ITEM_TIMING,
    enter: () => ({ keyframes: [ITEM_HIDDEN, ITEM_SHOWN], timing: ITEM_TIMING }),
    exit: () => ({ keyframes: [ITEM_SHOWN, ITEM_HIDDEN], timing: ITEM_TIMING }),
  });

  if (items.length === 0) {
    return null;
  }

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div ref={gridRef} className={cn("relative grid gap-2", getGridColsClass(gridCols))}>
        {items.map((item) => {
          const canRemove = !disabled && (!isItemRemovable || isItemRemovable(item));

          return (
            <div
              key={item.id}
              data-motion-key={item.id}
              className={cn(
                "flex items-center gap-2 px-3 py-1.5 bg-raised border rounded-md text-xs w-full",
                itemHeight,
              )}
            >
              <div className="border rounded-sm flex items-center justify-center bg-gray-4 shrink-0 size-5">
                {renderIcon(item)}
              </div>
              <div className="flex flex-col gap-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span
                    className="font-medium text-gray-12 text-left truncate max-w-[60px]"
                    title={item.name}
                  >
                    {renderPrimaryText(item)}
                  </span>
                  <span className="truncate z-auto">{renderBadge?.(item)}</span>
                </div>
                <span className="text-gray-9 text-2xs font-mono truncate">
                  {renderSecondaryText(item)}
                </span>
              </div>
              {canRemove ? (
                <button
                  type="button"
                  onClick={() => onRemoveItem(item.id)}
                  className="p-0.5 hover:bg-grayA-4 rounded-sm text-grayA-11 hover:text-gray-12 transition-colors shrink-0 ml-auto"
                  aria-label={`Remove ${renderPrimaryText(item)}`}
                >
                  <IconXmarkOutline12 />
                </button>
              ) : (
                <div
                  className="p-0.5 rounded-sm text-grayA-6 shrink-0 ml-auto opacity-50"
                  title="Cannot remove - inherited from selected role"
                >
                  <IconXmarkOutline12 />
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
