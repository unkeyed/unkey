"use client";

import { useSearchEndUsers } from "../hooks/use-search-end-users";
import { FormCombobox } from "@/components/ui/form-combobox";
import { TriangleWarning2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo } from "react";

type EndUserExternalIdFieldProps = {
  value: string | null;
  onChange: (identityId: string | null, externalId: string | null) => void;
  onSelectIdentity?: (identityId: string | null, externalId: string | null) => void;
  error?: string;
  disabled?: boolean;
};

export const EndUserExternalIdField = ({
  value,
  onChange,
  onSelectIdentity,
  error,
  disabled = false,
}: EndUserExternalIdFieldProps) => {
  const { searchValue, setSearchValue, searchResults, isSearching, trimmedSearchValue } =
    useSearchEndUsers();

  const exactMatch = searchResults.some(
    (identity) => identity.externalId.toLowerCase() === trimmedSearchValue.toLowerCase(),
  );

  const hasPartialMatches = searchResults.length > 0;

  // Create options from search results (identities)
  const options = searchResults.map((identity) => ({
    label: identity.externalId,
    value: identity.id,
    searchValue: identity.externalId,
    selectedLabel: <></>,
  }));

  // Don't show "Create" option if there's an exact match
  const createOption =
    trimmedSearchValue && !exactMatch && !isSearching
      ? {
          label: (
            <div className="flex items-center gap-2 w-full">
              <div
                className={cn(
                  "flex items-center rounded size-5 justify-center flex-shrink-0",
                  "bg-warningA-4",
                  "text-warning-11",
                )}
              >
                <TriangleWarning2 iconSize="sm-regular" />
              </div>
              <span className="text-[13px] text-gray-12 ">
                <span className="text-accent-10 font-normal">Create</span> "{trimmedSearchValue}"
              </span>
            </div>
          ),
          value: "__create_new__",
          selectedLabel: <></>,
          searchValue: trimmedSearchValue,
        }
      : null;

  const finalOptions = createOption ? [createOption, ...options] : options;

  const isLoading = isSearching && trimmedSearchValue.length > 0;

  return (
    <FormCombobox
      required
      label="External ID"
      description="ID of the end user in your system for billing attribution. Search for existing identities or create a new end user."
      options={finalOptions}
      key={value}
      value={value || ""}
      onChange={(e) => {
        setSearchValue(e.currentTarget.value);
      }}
      onSelect={(val) => {
        if (val === "__create_new__") {
          // When creating, pass null for identityId and the externalId
          onChange(null, trimmedSearchValue);
          if (onSelectIdentity) {
            onSelectIdentity(null, trimmedSearchValue);
          }
          return;
        }
        const identity = searchResults.find((id) => id.id === val);
        const identityId = identity?.id || null;
        const externalId = identity?.externalId || null;
        onChange(identityId, externalId);
        if (onSelectIdentity) {
          onSelectIdentity(identityId, externalId);
        }
      }}
      placeholder={
        <div className="flex w-full text-grayA-8 text-xs items-center py-2">Search or create External ID</div>
      }
      searchPlaceholder="Search External ID..."
      emptyMessage={
        trimmedSearchValue && !exactMatch ? (
          <div
            className={cn(
              "p-0 w-full transition-all duration-300 ease-in-out",
              "animate-in fade-in-0 slide-in-from-top-2",
            )}
          >
            <div className="px-3 py-3 w-full">
              <div className="flex gap-2 items-center justify-start">
                <div
                  className={cn(
                    "flex items-center rounded size-5 justify-center",
                    "bg-warningA-4",
                    "text-warning-11",
                    "transition-colors duration-200",
                  )}
                >
                  <TriangleWarning2 iconSize="sm-regular" />
                </div>
                <div className="font-medium text-[13px] leading-7 text-gray-12">
                  No matching identity found
                </div>
              </div>
            </div>
            <div className="w-full">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="px-4 w-full text-gray-11 text-[13px] leading-6 my-4 text-left">
              You can create a new end user with this{" "}
              <span className="font-medium">External ID</span> for billing purposes.
            </div>
          </div>
        ) : isLoading ? (
          <div className="px-3 py-3 text-gray-10 text-[13px] flex items-center gap-2">
            <div className="animate-spin h-3 w-3 border border-gray-6 border-t-gray-11 rounded-full" />
            Searching...
          </div>
        ) : (
          <div
            className={cn(
              "px-3 mt-2 text-gray-10 text-[13px]",
              "transition-all duration-200 ease-in-out",
              "animate-in fade-in-0",
            )}
          >
            No results found
          </div>
        )
      }
      variant="default"
      error={error}
      disabled={disabled || isLoading}
      loading={isLoading}
      title={isLoading ? "Searching for identities..." : undefined}
    />
  );
};