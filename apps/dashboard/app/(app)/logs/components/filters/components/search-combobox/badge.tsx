import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import type { SearchItem } from "./hooks";

type BadgeProps = {
  item: SearchItem;
  index: number;
  editingIndex: number;
  editInputRef: React.RefObject<HTMLInputElement>;
  onEditChange: (e: React.ChangeEvent<HTMLInputElement>, index: number) => void;
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
>) => (
  <Input
    ref={editInputRef}
    value={item.label + item.searchValue}
    onChange={(e) => onEditChange(e, index)}
    onBlur={onEditBlur}
    onKeyDown={onEditKeyDown}
    className="h-3 w-24 px-1 py-0 text-xs bg-transparent border-none focus:ring-0 focus:outline-none"
  />
);

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
    size="icon"
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
