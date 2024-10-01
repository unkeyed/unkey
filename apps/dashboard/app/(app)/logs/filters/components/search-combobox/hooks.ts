import { useEffect, useRef, useState } from "react";
import { KEYS, NO_ITEM_EDITING, OPTIONS } from "./constants";
import {
  type PickKeys,
  type QuerySearchParams,
  useLogSearchParams,
} from "../../query-state";

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
