import { SelectedItemsList } from "@/components/selected-item-list";
import { FormCombobox } from "@/components/ui/form-combobox";
import type { RoleKey } from "@/lib/trpc/routers/authorization/roles/connected-keys-and-perms";
import { Key2 } from "@unkey/icons";
import { useMemo, useState } from "react";
import { createKeyOptions } from "./create-key-options";
import { useFetchKeys } from "./hooks/use-fetch-keys";
import { useSearchKeys } from "./hooks/use-search-keys";

type KeyFieldProps = {
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
  disabled?: boolean;
  roleId?: string;
  assignedKeyDetails: RoleKey[];
};

export const KeyField = ({
  value,
  onChange,
  error,
  disabled = false,
  roleId,
  assignedKeyDetails,
}: KeyFieldProps) => {
  const [searchValue, setSearchValue] = useState("");
  const { keys, isFetchingNextPage, hasNextPage, loadMore, isLoading } = useFetchKeys();
  const { searchResults, isSearching } = useSearchKeys(searchValue);

  // Combine loaded keys with search results, prioritizing search when available
  const allKeys = useMemo(() => {
    if (searchValue.trim() && searchResults.length > 0) {
      // When searching, use search results
      return searchResults;
    }
    if (searchValue.trim() && searchResults.length === 0 && !isSearching) {
      // No search results found, filter from loaded keys as fallback
      const searchTerm = searchValue.toLowerCase().trim();
      return keys.filter(
        (key) =>
          key.id.toLowerCase().includes(searchTerm) || key.name?.toLowerCase().includes(searchTerm),
      );
    }
    // No search query, use all loaded keys
    return keys;
  }, [keys, searchResults, searchValue, isSearching]);

  // Don't show load more when actively searching
  const showLoadMore = !searchValue.trim() && hasNextPage;

  const baseOptions = createKeyOptions({
    keys: allKeys,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    roleId,
    loadMore,
  });

  const selectableOptions = useMemo(() => {
    return baseOptions.filter((option) => {
      // Always allow the load more option
      if (option.value === "__load_more__") {
        return true;
      }

      // Don't show already selected keys
      if (value.includes(option.value)) {
        return false;
      }

      // Find the key and check if it's already assigned to this role
      const key = allKeys.find((k) => k.id === option.value);
      if (!key) {
        return true;
      }

      // Filter out keys that already have this role assigned (if roleId provided)
      if (roleId) {
        return !key.roles.some((role) => role.id === roleId);
      }

      return true;
    });
  }, [baseOptions, allKeys, roleId, value]);

  const selectedKeys = useMemo(() => {
    return value
      .map((keyId) => {
        // check selectedKeysData (for pre-loaded edit data)
        const preLoadedKey = assignedKeyDetails.find((k) => k.id === keyId);
        if (preLoadedKey) {
          return {
            id: preLoadedKey.id,
            name: preLoadedKey.name,
          };
        }

        // check loaded keys (for newly added keys)
        const loadedKey = allKeys.find((k) => k.id === keyId);
        if (loadedKey) {
          return {
            id: loadedKey.id,
            name: loadedKey.name,
          };
        }

        // Third: fallback to ID-only display (ensures key is always removable)
        return {
          id: keyId,
          name: null,
        };
      })
      .filter((key): key is NonNullable<typeof key> => key !== undefined);
  }, [value, allKeys, assignedKeyDetails]);

  const handleRemoveKey = (keyId: string) => {
    onChange(value.filter((id) => id !== keyId));
  };

  const handleAddKey = (keyId: string) => {
    if (!value.includes(keyId)) {
      onChange([...value, keyId]);
    }
    setSearchValue("");
  };

  const isComboboxLoading = isLoading || (isSearching && searchValue.trim().length > 0);

  return (
    <div className="space-y-3">
      <FormCombobox
        optional
        label="Assign keys"
        description="Select keys from your workspace."
        options={selectableOptions}
        value=""
        onChange={(e) => setSearchValue(e.currentTarget.value)}
        onSelect={(val) => {
          if (val === "__load_more__") {
            return;
          }
          handleAddKey(val);
        }}
        placeholder={
          <div className="flex w-full text-grayA-8 text-[13px] gap-1.5 items-center py-2">
            Select keys
          </div>
        }
        searchPlaceholder="Search keys by name or ID..."
        emptyMessage={
          isComboboxLoading ? (
            <div className="px-3 py-3 text-gray-10 text-[13px] flex items-center gap-2">
              <div className="animate-spin h-3 w-3 border border-gray-6 border-t-gray-11 rounded-full" />
              {isSearching ? "Searching..." : "Loading keys..."}
            </div>
          ) : (
            <div className="px-3 py-3 text-gray-10 text-[13px]">No keys found</div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled || isLoading}
        loading={isComboboxLoading}
        title={
          isComboboxLoading
            ? isSearching && searchValue.trim()
              ? "Searching for keys..."
              : "Loading available keys..."
            : undefined
        }
      />

      <SelectedItemsList
        items={selectedKeys.map((k) => ({
          ...k,
          name: k.name ?? "Unnamed Key",
        }))}
        disabled={disabled}
        onRemoveItem={handleRemoveKey}
        renderIcon={() => <Key2 size="sm-regular" className="text-grayA-11" />}
        enableTransitions
        renderPrimaryText={(key) =>
          key.id.length > 15 ? `${key.id.slice(0, 8)}...${key.id.slice(-4)}` : key.id
        }
        renderSecondaryText={(key) => key.name || "Unnamed Key"}
        itemHeight="h-12"
      />
    </div>
  );
};
