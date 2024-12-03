import { Button } from "@/components/ui/button";
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { CheckCircle, Search } from "lucide-react";
import { useCallback, useRef, useState } from "react";

import { useLogSearchParams } from "../../../../query-state";
import { ComboboxBadge } from "./badge";
import {
  KEYS,
  NO_ITEM_EDITING,
  OPTIONS,
  OPTION_EXPLANATIONS,
  PLACEHOLDER_TEXT,
} from "./constants";
import {
  type Option,
  type SearchItem,
  useDeleteFromSelection,
  useFocusOnBadge,
  useListenEscapeKey,
  useSelectComboboxItems,
} from "./hooks";

export function SearchCombobox() {
  const [open, setOpen] = useState<boolean>(false);
  const selectedItemWrapperRef = useRef<HTMLDivElement>(null);
  const [currentFocusedItemIndex, setCurrentFocusedItemIndex] =
    useState<number>(NO_ITEM_EDITING);

  const { selectedItems, setSelectedItems } = useSelectComboboxItems();
  const { editInputRef } = useFocusOnBadge(currentFocusedItemIndex);
  const { setSearchParams } = useLogSearchParams();

  const handlePressEscape = useCallback(() => {
    setOpen(false);
    setCurrentFocusedItemIndex(NO_ITEM_EDITING);
  }, []);

  useListenEscapeKey(handlePressEscape);

  const handleRemove = (item: SearchItem) => {
    setSelectedItems((prevState) =>
      prevState.filter((selected) => selected.value !== item.value)
    );
    setSearchParams({ [item.value]: null });
  };
  useDeleteFromSelection(selectedItems, handleRemove, selectedItemWrapperRef);

  const handleSelect = (item: Option) => {
    setSelectedItems((prevItems) => {
      if (!prevItems.some((selected) => selected.value === item.value)) {
        const newItems = [...prevItems, { ...item, searchValue: "" }];
        // Schedule the edit of the last item after the state update
        setTimeout(() => handleFocusOnClick(newItems.length - 1), 0);
        return newItems;
      }
      return prevItems;
    });
  };

  const handleFocusOnClick = (index: number) => {
    setCurrentFocusedItemIndex(index);
  };

  const handleEditChange = (item: SearchItem, index: number) => {
    setSelectedItems((prevItems) => {
      const newSearchValue = item.searchValue;
      const newItems = [...prevItems];
      newItems[index] = { ...item, searchValue: newSearchValue };

      setSearchParams({
        [item.value]: newSearchValue.length > 0 ? newSearchValue : null,
      });

      return newItems;
    });
  };

  const handleEditBlur = () => {
    setCurrentFocusedItemIndex(NO_ITEM_EDITING);
  };

  const handleEditKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === KEYS.ENTER) {
      setCurrentFocusedItemIndex(NO_ITEM_EDITING);
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
                  <ComboboxBadge
                    editInputRef={editInputRef}
                    editingIndex={currentFocusedItemIndex}
                    onEditBlur={handleEditBlur}
                    onEditChange={handleEditChange}
                    onEditKeyDown={handleEditKeyDown}
                    onFocus={handleFocusOnClick}
                    onRemove={handleRemove}
                    index={index}
                    item={item}
                    key={item.value}
                  />
                ))}
              </div>
            ) : (
              PLACEHOLDER_TEXT
            )}
          </div>
        </Button>
      </PopoverTrigger>
      {/* Forces popover content to strech relative to its parent */}
      <PopoverContent
        className="w-[--radix-popover-trigger-width] max-h-[--radix-popover-content-available-height] p-0"
        ref={selectedItemWrapperRef}
      >
        <Command>
          <CommandList>
            <CommandGroup>
              {OPTIONS.map((framework) => (
                <CommandItem
                  key={framework.value}
                  value={framework.value}
                  onSelect={() => {
                    //Focuses on clicked item if exists
                    setCurrentFocusedItemIndex(
                      selectedItems.findIndex(
                        (i) => i.value === framework.value
                      )
                    );
                    handleSelect(framework);
                  }}
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
