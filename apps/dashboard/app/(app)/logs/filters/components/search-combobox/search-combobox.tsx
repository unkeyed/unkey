import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { CheckCircle, Search, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  NO_ITEM_EDITING,
  KEYS,
  OPTIONS,
  PLACEHOLDER_TEXT,
  OPTION_EXPLANATIONS,
} from "./constants";

type Option = {
  value: string;
  label: string;
};
type SearchItem = Option & { searchValue: string };

export function SearchCombobox() {
  const [open, setOpen] = useState<boolean>(false);
  const [selectedItems, setSelectedItems] = useState<SearchItem[]>([]);
  const [editingIndex, setEditingIndex] = useState<number>(NO_ITEM_EDITING);
  const editInputRef = useRef<HTMLInputElement>(null);

  // Focuses on a badge
  useEffect(() => {
    if (editingIndex !== NO_ITEM_EDITING && editInputRef.current) {
      editInputRef.current.focus();
    }
  }, [editingIndex]);

  // Let's you escape from double-clicked badge after editting
  useEffect(() => {
    const handleEscKey = (event: KeyboardEvent) => {
      if (event.key === KEYS.ESCAPE) {
        setOpen(false);
        setEditingIndex(NO_ITEM_EDITING);
      }
    };

    document.addEventListener("keydown", handleEscKey);

    return () => {
      document.removeEventListener("keydown", handleEscKey);
    };
  }, []);

  const handleSelect = (item: Option) => {
    setSelectedItems((prevItems) => {
      if (!prevItems.some((selected) => selected.value === item.value)) {
        const newItems = [...prevItems, { ...item, searchValue: "" }];
        // Schedule the edit of the last item after the state update
        setTimeout(() => handleEdit(newItems.length - 1), 0);
        return newItems;
      }
      return prevItems;
    });
  };

  const handleRemove = (item: Option) => {
    setSelectedItems(
      selectedItems.filter((selected) => selected.value !== item.value)
    );
  };

  const handleEdit = (index: number) => {
    setEditingIndex(index);
  };

  const handleEditChange = (
    e: React.ChangeEvent<HTMLInputElement>,
    index: number
  ) => {
    const value = e.target.value;
    setSelectedItems((prevItems) => {
      const item = prevItems[index];
      const undeletablePart =
        OPTIONS.find((o) => o.value === item.value)?.label || "";

      if (value.startsWith(undeletablePart)) {
        const newSearchValue = value.slice(undeletablePart.length);
        const newItems = [...prevItems];
        newItems[index] = { ...item, searchValue: newSearchValue };
        return newItems;
      }

      return prevItems;
    });
  };

  const handleEditBlur = () => {
    setEditingIndex(NO_ITEM_EDITING);
  };

  const handleEditKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === KEYS.ENTER) {
      setEditingIndex(NO_ITEM_EDITING);
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="min-w-[330px] justify-start hover:bg-current! hover:text-current! text-content-subtle"
        >
          <div className="flex gap-2 flex-nowrap items-center w-fit">
            <Search size={16} className="text-muted-foreground" />
            {selectedItems.length > 0 ? (
              <div className="flex gap-1 flex-wrap items-center">
                {selectedItems.map((item, index) => (
                  <Badge
                    key={item.value}
                    variant="secondary"
                    className={cn(
                      "flex items-center pr-1 w-fit",
                      "focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-3",
                      "transition-shadow duration-200",
                      "hover:bg-secondary/80",
                      "group-hover:bg-secondary" // Preserve badge color when main button is hovered
                    )}
                  >
                    {editingIndex === index ? (
                      <Input
                        ref={editInputRef}
                        value={item.label + item.searchValue}
                        onChange={(e) => {
                          handleEditChange(e, index);
                        }}
                        onBlur={handleEditBlur}
                        onKeyDown={handleEditKeyDown}
                        className="h-3 w-24 px-1 py-0 text-xs bg-transparent border-none focus:ring-0 focus:outline-none"
                      />
                    ) : (
                      <div className="max-w-[150px]">
                        <span
                          className="block truncate cursor-text"
                          onDoubleClick={(e) => {
                            e.stopPropagation();
                            handleEdit(index);
                          }}
                          title={item.label + item.searchValue}
                        >
                          <span
                            className={cn(
                              item.searchValue ? "font-medium" : ""
                            )}
                          >
                            {item.label}
                          </span>
                          <span>{item.searchValue}</span>
                        </span>
                      </div>
                    )}
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="flex-shrink-0 w-4 h-4 min-w-[16px] hover:bg-secondary/80"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleRemove(item);
                      }}
                    >
                      <X className="w-3 h-3" />
                    </Button>
                  </Badge>
                ))}
              </div>
            ) : (
              PLACEHOLDER_TEXT
            )}
          </div>
        </Button>
      </PopoverTrigger>
      {/* Forces popover content to strech relative to its parent */}
      <PopoverContent className="w-[--radix-popover-trigger-width] max-h-[--radix-popover-content-available-height] p-0">
        <Command>
          <CommandList>
            <CommandGroup>
              {OPTIONS.map((framework) => (
                <CommandItem
                  key={framework.value}
                  value={framework.value}
                  onSelect={() => handleSelect(framework)}
                  className="group"
                >
                  <div className="flex gap-2 items-center ">
                    <div className="px-2 bg-gray-200 rounded-md group-hover:bg-gray-300 font-medium text-[13px]">
                      {framework.label}
                    </div>
                    <div className="hidden group-hover:block text-content-subtle">
                      {OPTION_EXPLANATIONS[framework.value]}
                    </div>
                  </div>
                  <CheckCircle
                    className={cn(
                      "ml-auto h-4 w-4",
                      selectedItems.some(
                        (item) => item.value === framework.value
                      )
                        ? "opacity-100"
                        : "opacity-0"
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
