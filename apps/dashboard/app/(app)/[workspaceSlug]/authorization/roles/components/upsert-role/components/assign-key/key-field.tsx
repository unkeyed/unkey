import { SelectedItemsList } from "@/components/selected-item-list";
import { FormCombobox } from "@/components/ui/form-combobox";
import type { RoleKey } from "@/lib/trpc/routers/authorization/roles/connected-keys-and-perms";
import { Key2 } from "@unkey/icons";
import { useMemo, useState } from "react";
import { useRoleLimits } from "../../../table/hooks/use-role-limits";
import { RoleWarningCallout } from "../warning-callout";
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

  const { calculateLimits } = useRoleLimits(roleId);
  const { hasKeyWarning, totalKeys } = calculateLimits(value);

  const { keys, isFetchingNextPage, hasNextPage, loadMore, isLoading } =
    useFetchKeys();
  const { searchResults, isSearching } = useSearchKeys(searchValue);

  const allKeys = useMemo(() => {
    if (searchValue.trim() && searchResults.length > 0) {
      return searchResults;
    }

    if (searchValue.trim() && searchResults.length === 0 && !isSearching) {
      const searchTerm = searchValue.toLowerCase().trim();
      return keys.filter(
        (key) =>
          key.id.toLowerCase().includes(searchTerm) ||
          key.name?.toLowerCase().includes(searchTerm)
      );
    }

    return keys;
  }, [keys, searchResults, searchValue, isSearching]);

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
      if (option.value === "__load_more__") {
        return true;
      }

      if (value.includes(option.value)) {
        return false;
      }

      const key = allKeys.find((k) => k.id === option.value);
      if (!key) {
        return true;
      }

      if (roleId) {
        return !key.roles.some((role) => role.id === roleId);
      }

      return true;
    });
  }, [baseOptions, allKeys, roleId, value]);

  const selectedKeys = useMemo(() => {
    return value
      .map((keyId) => {
        const preLoadedKey = assignedKeyDetails.find((k) => k.id === keyId);
        if (preLoadedKey) {
          return {
            id: preLoadedKey.id,
            name: preLoadedKey.name,
          };
        }

        const loadedKey = allKeys.find((k) => k.id === keyId);
        if (loadedKey) {
          return {
            id: loadedKey.id,
            name: loadedKey.name,
          };
        }

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

  const isComboboxLoading =
    isLoading || (isSearching && searchValue.trim().length > 0);

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
            <div className="px-3 py-3 text-gray-10 text-[13px]">
              No keys found
            </div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled || isLoading || hasKeyWarning}
        loading={isComboboxLoading}
        title={
          isComboboxLoading
            ? isSearching && searchValue.trim()
              ? "Searching for keys..."
              : "Loading available keys..."
            : undefined
        }
      />
      {hasKeyWarning ? (
        <RoleWarningCallout count={totalKeys} type="keys" />
      ) : (
        <SelectedItemsList
          items={selectedKeys.map((k) => ({
            ...k,
            name: k.name ?? "Unnamed Key",
          }))}
          disabled={disabled}
          onRemoveItem={handleRemoveKey}
          renderIcon={() => (
            <Key2 iconSize="sm-regular" className="text-grayA-11" />
          )}
          enableTransitions
          renderPrimaryText={(key) =>
            key.id.length > 15
              ? `${key.id.slice(0, 8)}...${key.id.slice(-4)}`
              : key.id
          }
          renderSecondaryText={(key) => key.name || "Unnamed Key"}
          itemHeight="h-12"
        />
      )}
    </div>
  );
};
