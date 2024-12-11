import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { OPTIONS } from "./constants";
import type { SearchItem } from "./hooks";

type BadgeProps = {
  item: SearchItem;
  index: number;
  editingIndex: number;
  editInputRef: React.RefObject<HTMLInputElement>;
  onEditChange: (item: SearchItem, index: number) => void;
  onEditBlur: () => void;
  onEditKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onFocus: (index: number) => void;
  onRemove: (item: SearchItem) => void;
};

const ActiveBadgeContent = ({
  item,
  index,
  editInputRef,
  onEditChange,
  onEditBlur,
  onEditKeyDown,
}: Pick<
  BadgeProps,
  "item" | "index" | "editInputRef" | "onEditChange" | "onEditBlur" | "onEditKeyDown"
>) => {
  const [value, setValue] = useState(item.label + item.searchValue);
  const timeoutIdRef = useRef<ReturnType<typeof setTimeout>>();

  const debouncedOnEditChange = useCallback(
    (newItem: typeof item, idx: number) => {
      clearTimeout(timeoutIdRef.current);
      timeoutIdRef.current = setTimeout(() => {
        onEditChange(newItem, idx);
      }, 300);
    },
    [onEditChange],
  );

  const internalHandleEditChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      const undeletablePart = OPTIONS.find((o) => o.value === item.value)?.label || "";
      const newSearchValue = value.slice(undeletablePart.length);
      setValue(item.label + newSearchValue);

      debouncedOnEditChange(
        {
          value: item.value,
          label: item.label,
          searchValue: newSearchValue,
        },
        index,
      );
    },
    [item.value, item.label, index, debouncedOnEditChange],
  );

  useEffect(() => {
    return () => {
      clearTimeout(timeoutIdRef.current);
    };
  }, []);

  return (
    <Input
      ref={editInputRef}
      value={value}
      onChange={internalHandleEditChange}
      onBlur={() => {
        // Clear any pending timeout and call onEditChange immediately on blur
        clearTimeout(timeoutIdRef.current);
        onEditChange(
          {
            value: item.value,
            label: item.label,
            searchValue: value.slice(item.label.length),
          },
          index,
        );
        onEditBlur?.();
      }}
      onKeyDown={onEditKeyDown}
      className="h-3 w-24 px-1 py-0 text-xs bg-transparent border-none focus:ring-0 focus:outline-none"
    />
  );
};
const PassiveBadgeContent = ({
  item,
  index,
  onFocus: handleFocusOnClick,
}: Pick<BadgeProps, "item" | "index" | "onFocus">) => (
  <div className="max-w-[150px]">
    <span
      className="block truncate cursor-text"
      onDoubleClick={(e) => {
        e.stopPropagation();
        handleFocusOnClick(index);
      }}
      title={item.label + item.searchValue}
    >
      <span className={cn(item.searchValue ? "font-medium" : "")}>{item.label}</span>
      <span>{item.searchValue}</span>
    </span>
  </div>
);

const RemoveButton = ({ item, onRemove }: Pick<BadgeProps, "item" | "onRemove">) => (
  <Button
    type="button"
    variant="ghost"
    shape="square"
    className="flex-shrink-0 w-4 h-4 min-w-[16px] hover:bg-secondary/80"
    onClick={(e) => {
      e.stopPropagation();
      onRemove(item);
    }}
  >
    <X className="w-3 h-3" />
  </Button>
);

export const ComboboxBadge = ({
  item,
  index,
  editingIndex,
  editInputRef,
  onEditChange: handleEditChange,
  onEditBlur: handleEditBlur,
  onEditKeyDown: handleEditKeyDown,
  onFocus: handleFocusOnClick,
  onRemove: handleRemove,
}: BadgeProps) => (
  <Badge
    key={item.value}
    variant="secondary"
    className={cn(
      "flex items-center pr-1 w-fit z-3",
      "focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-3",
      "transition-shadow duration-200",
      "hover:bg-secondary/80",
    )}
  >
    {editingIndex === index ? (
      <ActiveBadgeContent
        item={item}
        index={index}
        editInputRef={editInputRef}
        onEditChange={handleEditChange}
        onEditBlur={handleEditBlur}
        onEditKeyDown={handleEditKeyDown}
      />
    ) : (
      <PassiveBadgeContent item={item} index={index} onFocus={handleFocusOnClick} />
    )}
    <RemoveButton item={item} onRemove={handleRemove} />
  </Badge>
);
