import { useEffect, useRef, useState } from "react";
import {
  type PickKeys,
  type QuerySearchParams,
  useLogSearchParams,
} from "../../../../query-state";
import { KEYS, NO_ITEM_EDITING, OPTIONS } from "./constants";

export const useFocusOnBadge = (currentFocusedItemIndex: number) => {
  const editInputRef = useRef<HTMLInputElement>(null);

  // Focuses on a badge
  useEffect(() => {
    if (currentFocusedItemIndex !== NO_ITEM_EDITING && editInputRef.current) {
      editInputRef.current.focus();
    }
  }, [currentFocusedItemIndex]);

  return { editInputRef };
};

export type Option = {
  value: PickKeys<QuerySearchParams, "host" | "requestId" | "path" | "method">;
  label: string;
};
export type SearchItem = Option & { searchValue: string };

export const useSelectComboboxItems = () => {
  const [selectedItems, setSelectedItems] = useState<SearchItem[]>([]);
  const { searchParams, setSearchParams } = useLogSearchParams();

  // biome-ignore lint/correctness/useExhaustiveDependencies: When "setSelectedItems" included hook does too many renders
  useEffect(() => {
    const initialItems = OPTIONS.filter(
      (option) => searchParams[option.value]
    ).map((option) => ({
      ...option,
      searchValue: searchParams[option.value] as string,
    }));
    setSelectedItems(initialItems);
  }, []);

  // biome-ignore lint/correctness/useExhaustiveDependencies: When setSearchParams included component does too many retries
  useEffect(() => {
    setSearchParams(
      selectedItems.reduce((params, item) => {
        if (item.searchValue) {
          params[item.value] = item.searchValue;
        }
        return params;
      }, {} as Partial<QuerySearchParams>)
    );
  }, [selectedItems]);

  return { selectedItems, setSelectedItems } as const;
};

export const useListenEscapeKey = (cb: () => void) => {
  // Let's you escape from double-clicked badge after editting
  useEffect(() => {
    const handleEscKey = (event: KeyboardEvent) => {
      if (event.key === KEYS.ESCAPE) {
        cb();
      }
    };

    document.addEventListener("keydown", handleEscKey);

    return () => {
      document.removeEventListener("keydown", handleEscKey);
    };
  }, [cb]);
};

const DELETE_KEYS = {
  DELETE: "Delete",
  BACKSPACE: "Backspace",
} as const;

export const useDeleteFromSelection = (
  selectedItems: SearchItem[],
  onRemoveFromSelectedItems: (item: SearchItem) => void,
  elementRef: React.RefObject<HTMLElement>
) => {
  useEffect(() => {
    const handleDeleteKey = (event: KeyboardEvent) => {
      if (document.activeElement !== elementRef.current) return;

      if (
        event.key === DELETE_KEYS.DELETE ||
        event.key === DELETE_KEYS.BACKSPACE
      ) {
        event.preventDefault();
        const lastItem = selectedItems?.at(-1);
        if (selectedItems.length > 0 && lastItem) {
          onRemoveFromSelectedItems(lastItem);
        }
      }
    };

    document.addEventListener("keydown", handleDeleteKey);
    return () => {
      document.removeEventListener("keydown", handleDeleteKey);
    };
  }, [selectedItems, elementRef]);
};
