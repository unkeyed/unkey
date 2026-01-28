import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import { AnimatePresence, motion } from "framer-motion";

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
  if (items.length === 0) {
    return null;
  }

  const ItemComponent = enableTransitions ? motion.div : "div";

  return (
    <div className={cn("space-y-2", className)}>
      <div className={cn("grid gap-2", getGridColsClass(gridCols))}>
        <AnimatePresence mode="popLayout">
          {items.map((item) => {
            const canRemove = !disabled && (!isItemRemovable || isItemRemovable(item));
            const itemProps = enableTransitions
              ? {
                  layout: true,
                  initial: { opacity: 0, scale: 0.8, y: -10 },
                  animate: { opacity: 1, scale: 1, y: 0 },
                  exit: { opacity: 0, scale: 0.8, y: -10 },
                  transition: {
                    type: "spring" as const,
                    stiffness: 500,
                    damping: 30,
                    mass: 0.8,
                  },
                }
              : {};

            return (
              <ItemComponent
                key={item.id}
                className={cn(
                  "flex items-center gap-2 px-3 py-1.5 bg-white dark:bg-black border border-gray-5 rounded-md text-xs w-full",
                  itemHeight,
                )}
                {...itemProps}
              >
                <div className="border rounded flex items-center justify-center border-grayA-4 bg-gray-4 flex-shrink-0 size-5">
                  {renderIcon(item)}
                </div>
                <div className="flex flex-col gap-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span
                      className="font-medium text-accent-12 text-left truncate max-w-[60px]"
                      title={item.name}
                    >
                      {renderPrimaryText(item)}
                    </span>
                    <span className="truncate z-auto">{renderBadge?.(item)}</span>
                  </div>
                  <span className="text-accent-9 text-[11px] font-mono truncate">
                    {renderSecondaryText(item)}
                  </span>
                </div>
                {canRemove ? (
                  <button
                    type="button"
                    onClick={() => onRemoveItem(item.id)}
                    className="p-0.5 hover:bg-grayA-4 rounded text-grayA-11 hover:text-accent-12 transition-colors flex-shrink-0 ml-auto"
                    aria-label={`Remove ${renderPrimaryText(item)}`}
                  >
                    <XMark iconSize="sm-regular" />
                  </button>
                ) : (
                  <div
                    className="p-0.5 rounded text-grayA-6 flex-shrink-0 ml-auto opacity-50"
                    title="Cannot remove - inherited from selected role"
                  >
                    <XMark iconSize="sm-regular" />
                  </div>
                )}
              </ItemComponent>
            );
          })}
        </AnimatePresence>
      </div>
    </div>
  );
}
