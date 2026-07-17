import { useCreateIdentity } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/hooks/use-create-identity";
import { FormCombobox } from "@/components/ui/form-combobox";
import { useIdentities } from "@/lib/identities-query";
import { identityExternalIdSchema } from "@/lib/schemas/identity";
import { getErrorMessage } from "@/lib/unkey-client";
import type { Identity } from "@unkey/api/models/components";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useDeferredValue, useMemo, useState } from "react";
import { createIdentityOptions } from "./create-identity-options";

type ExternalIdFieldProps = {
  value: string | null;
  onChange: (identityId: string | null, externalId: string | null) => void;
  error?: string;
  disabled?: boolean;
  currentIdentity?: {
    id: string;
    externalId: string;
    meta?: Identity["meta"];
  };
};

export const ExternalIdField = ({
  value,
  onChange,
  error,
  disabled = false,
  currentIdentity,
}: ExternalIdFieldProps) => {
  const [searchValue, setSearchValue] = useState("");

  const trimmedSearchValue = searchValue.trim();
  const externalIdValidation = identityExternalIdSchema.safeParse(trimmedSearchValue);
  const externalIdError = externalIdValidation.success
    ? undefined
    : externalIdValidation.error.issues.at(0)?.message;
  const deferredSearch = useDeferredValue(trimmedSearchValue);
  const { identities, isFetchingNextPage, hasNextPage, fetchNextPage, isLoading } = useIdentities({
    onError: (error) => {
      toast.error("Failed to Load Identities", {
        description: getErrorMessage(error, "We were unable to load identities. Please try again."),
      });
    },
  });
  const { identities: searchResults, isLoading: isSearchLoading } = useIdentities({
    search: deferredSearch,
    enabled: deferredSearch.length > 0,
    onError: (error) => {
      toast.error("Failed to Search Identities", {
        description: getErrorMessage(
          error,
          "We were unable to search identities. Please try again.",
        ),
      });
    },
  });
  const isSearching =
    trimmedSearchValue !== deferredSearch || (deferredSearch.length > 0 && isSearchLoading);
  const loadMore = () => {
    fetchNextPage().catch((error: unknown) => {
      console.error("Failed to load more identities", error);
    });
  };

  const createIdentity = useCreateIdentity((data) => {
    setSearchValue("");
    onChange(data.identityId, data.externalId);
  });

  // Combine loaded identities with search results, prioritizing search when available
  const allIdentities = useMemo(() => {
    if (trimmedSearchValue && searchResults.length > 0) {
      // When searching, use search results
      return searchResults;
    }
    if (trimmedSearchValue && searchResults.length === 0 && !isSearching) {
      // No search results found, filter from loaded identities as fallback
      const searchTerm = trimmedSearchValue.toLowerCase();
      return identities.filter((identity) =>
        identity.externalId.toLowerCase().includes(searchTerm),
      );
    }
    // No search query, use all loaded identities
    return identities;
  }, [identities, searchResults, trimmedSearchValue, isSearching]);

  // Ensure current identity is always available in the options
  const allIdentitiesWithCurrent = useMemo(() => {
    if (!currentIdentity || !value) {
      return allIdentities;
    }

    // Check if current identity is already in the list
    const currentExists = allIdentities.some((identity) => identity.id === currentIdentity.id);

    if (currentExists) {
      return allIdentities;
    }

    return [
      {
        id: currentIdentity.id,
        externalId: currentIdentity.externalId,
        meta: currentIdentity.meta || {},
      },
      ...allIdentities,
    ];
  }, [allIdentities, currentIdentity, value]);

  const selectedExternalId = useMemo(() => {
    if (!value) {
      return undefined;
    }
    return allIdentitiesWithCurrent.find((id) => id.id === value)?.externalId;
  }, [allIdentitiesWithCurrent, value]);

  const handleCreateIdentity = () => {
    if (externalIdValidation.success) {
      createIdentity.mutate({
        externalId: externalIdValidation.data,
      });
    }
  };

  const exactMatch = allIdentitiesWithCurrent.some(
    (id) => id.externalId.toLowerCase() === trimmedSearchValue.toLowerCase(),
  );

  const hasPartialMatches = allIdentitiesWithCurrent.length > 0;

  // Don't show load more when actively searching
  const showLoadMore = !trimmedSearchValue && hasNextPage;

  const baseOptions = createIdentityOptions({
    identities: allIdentitiesWithCurrent,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    loadMore,
  });

  const createOption =
    externalIdValidation.success && !exactMatch && hasPartialMatches && !isSearching
      ? {
          label: (
            <div className="flex items-center gap-2 w-full">
              <div
                className={cn(
                  "flex items-center rounded-sm size-5 justify-center shrink-0",
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

  const options = createOption ? [createOption, ...baseOptions] : baseOptions;

  const isComboboxLoading = isLoading || (isSearching && trimmedSearchValue.length > 0);

  return (
    <FormCombobox
      requirement="optional"
      label="External ID"
      description="ID of the user/workspace in your system for key attribution."
      options={options}
      key={value}
      value={value || ""}
      onChange={(e) => {
        setSearchValue(e.currentTarget.value);
      }}
      onSelect={(val) => {
        if (val === "__load_more__") {
          loadMore();
          return;
        }
        if (val === "__create_new__") {
          handleCreateIdentity();
          return;
        }
        const identity = allIdentitiesWithCurrent.find((id) => id.id === val);
        setSearchValue("");
        onChange(identity?.id || null, identity?.externalId || null);
      }}
      placeholder={
        <div className="flex w-full text-grayA-8 text-xs items-center py-2">Select External ID</div>
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
                    "flex items-center rounded-sm size-5 justify-center",
                    "bg-warningA-4",
                    "text-warning-11",
                    "transition-colors duration-200",
                  )}
                >
                  <TriangleWarning2 iconSize="sm-regular" />
                </div>
                <div className="font-medium text-[13px] leading-7 text-gray-12">
                  {externalIdValidation.success ? "External ID not found" : "Invalid external ID"}
                </div>
              </div>
            </div>
            <div className="w-full">
              <div className="h-px bg-grayA-3 w-full" />
            </div>
            {externalIdValidation.success ? (
              <>
                <div className="px-4 w-full text-gray-11 text-[13px] leading-6 my-4 text-left">
                  You can create a new identity with this{" "}
                  <span className="font-medium">External ID</span> and connect it{" "}
                  <span className="font-medium">immediately</span>.
                </div>
                <div className="w-full px-4 pb-4">
                  <Button
                    type="button"
                    variant="primary"
                    size="xlg"
                    className={cn(
                      "rounded-lg w-full",
                      "transition-all duration-200 ease-in-out",
                      "hover:scale-[1.02] active:scale-[0.98]",
                    )}
                    onClick={handleCreateIdentity}
                    loading={createIdentity.isLoading}
                    disabled={createIdentity.isLoading || disabled}
                  >
                    Create
                  </Button>
                </div>
              </>
            ) : (
              <div
                role="alert"
                className="px-4 w-full text-error-11 text-[13px] leading-6 my-4 text-left"
              >
                {externalIdError}
              </div>
            )}
          </div>
        ) : isComboboxLoading ? (
          <div className="px-3 py-3 text-gray-10 text-[13px] flex items-center gap-2">
            <div className="animate-spin h-3 w-3 border border-gray-6 border-t-gray-11 rounded-full" />
            {isSearching ? "Searching..." : "Loading identities..."}
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
      loading={isComboboxLoading}
      title={
        isComboboxLoading
          ? isSearching && trimmedSearchValue
            ? "Searching for identities..."
            : "Loading available identities..."
          : undefined
      }
      copyValue={selectedExternalId}
    />
  );
};
