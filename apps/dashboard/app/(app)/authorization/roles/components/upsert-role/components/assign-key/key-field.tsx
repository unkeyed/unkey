import { FormCombobox } from "@/components/ui/form-combobox";
import type { RoleKey } from "@/lib/trpc/routers/authorization/roles/connected-keys-and-perms";
import { Key2, XMark } from "@unkey/icons";
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
  const { keys, isFetchingNextPage, hasNextPage, loadMore } = useFetchKeys();
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
        // First: check selectedKeysData (for pre-loaded edit data)
        const preLoadedKey = assignedKeyDetails.find((k) => k.id === keyId);
        if (preLoadedKey) {
          return {
            id: preLoadedKey.id,
            name: preLoadedKey.name,
          };
        }

        // Second: check loaded keys (for newly added keys)
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
          // Add the selected key to the array
          if (!value.includes(val)) {
            onChange([...value, val]);
          }
          // Clear search after selection
          setSearchValue("");
        }}
        placeholder={
          <div className="flex w-full text-grayA-8 text-[13px] gap-1.5 items-center py-2">
            Select keys
          </div>
        }
        searchPlaceholder="Search keys by name or ID..."
        emptyMessage={
          isSearching ? (
            <div className="px-3 py-3 text-gray-10 text-[13px]">Searching...</div>
          ) : (
            <div className="px-3 py-3 text-gray-10 text-[13px]">No keys found</div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled}
      />

      {/* Selected Keys Display */}
      {selectedKeys.length > 0 && (
        <div className="space-y-2">
          <div className="grid grid-cols-2 gap-2 max-w-[400px]">
            {selectedKeys.map((key) => (
              <div
                key={key.id}
                className="flex items-center gap-2 px-3 py-1.5 bg-white dark:bg-black border border-gray-5 rounded-md text-xs h-12 w-full"
              >
                <div className="border rounded-full flex items-center justify-center border-grayA-6 size-4 flex-shrink-0">
                  <Key2 size="sm-regular" className="text-grayA-11" />
                </div>
                <div className="flex flex-col gap-0.5 min-w-0">
                  <span className="font-medium text-accent-12 truncate text-xs">
                    {key.id.length > 15 ? `${key.id.slice(0, 8)}...${key.id.slice(-4)}` : key.id}
                  </span>
                  <span className="text-accent-9 text-[11px] font-mono truncate">
                    {key.name || "Unnamed Key"}
                  </span>
                </div>
                {!disabled && (
                  <button
                    type="button"
                    onClick={() => handleRemoveKey(key.id)}
                    className="p-0.5 hover:bg-grayA-4 rounded text-grayA-11 hover:text-accent-12 transition-colors flex-shrink-0 ml-auto"
                    aria-label={`Remove ${key.name || key.id}`}
                  >
                    <XMark size="sm-regular" />
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
