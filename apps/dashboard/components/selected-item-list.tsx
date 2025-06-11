import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";

interface BaseItem {
  id: string;
  name?: string | null;
}

interface SelectedItemsListProps<T extends BaseItem> {
  items: T[];
  disabled?: boolean;
  onRemoveItem: (id: string) => void;
  renderIcon: (item: T) => React.ReactNode;
  renderPrimaryText: (item: T) => string;
  renderSecondaryText: (item: T) => string;
  renderBadge?: (item: T) => React.ReactNode;
  className?: string;
  gridCols?: 1 | 2 | 3 | 4;
  itemHeight?: string;
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
  renderIcon,
  renderPrimaryText,
  renderSecondaryText,
  renderBadge,
  className,
  gridCols = 2,
  itemHeight,
}: SelectedItemsListProps<T>) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className={cn("space-y-2", className)}>
      <div className={cn("grid gap-2", getGridColsClass(gridCols))}>
        {items.map((item) => (
          <div
            key={item.id}
            className={cn(
              "flex items-center gap-2 px-3 py-1.5 bg-white dark:bg-black border border-gray-5 rounded-md text-xs w-full",
              itemHeight,
            )}
          >
            {renderIcon(item)}
            <div className="flex flex-col gap-1 min-w-0">
              <div className="flex items-center gap-2">
                <span
                  className="font-medium text-accent-12 text-left truncate max-w-[60px]"
                  title={item.name}
                >
                  {renderPrimaryText(item)}
                </span>
                <span className="truncate">{renderBadge?.(item)}</span>
              </div>
              <span className="text-accent-9 text-[11px] font-mono truncate">
                {renderSecondaryText(item)}
              </span>
            </div>
            {!disabled && (
              <button
                type="button"
                onClick={() => onRemoveItem(item.id)}
                className="p-0.5 hover:bg-grayA-4 rounded text-grayA-11 hover:text-accent-12 transition-colors flex-shrink-0 ml-auto"
                aria-label={`Remove ${renderPrimaryText(item)}`}
              >
                <XMark size="sm-regular" />
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
