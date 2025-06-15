import { useCreateIdentity } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-create-identity";
import { useFetchIdentities } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities";
import { createIdentityOptions } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities/create-identity-options";
import { FormCombobox } from "@/components/ui/form-combobox";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, Loading } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState, useTransition } from "react";
import { useSearchIdentities } from "./use-search-identities";

type ExternalIdFieldProps = {
  value: string | null;
  onChange: (identityId: string | null, externalId: string | null) => void;
  error?: string;
  disabled?: boolean;
};

export const ExternalIdField = ({
  value,
  onChange,
  error,
  disabled = false,
}: ExternalIdFieldProps) => {
  const [searchValue, setSearchValue] = useState("");
  const [isPending, startTransition] = useTransition();
  const { identities, isFetchingNextPage, hasNextPage, loadMore } = useFetchIdentities();
  const { searchResults, isSearching } = useSearchIdentities(searchValue);

  const createIdentity = useCreateIdentity((data) => {
    onChange(data.identityId, data.externalId);
  });

  // Combine loaded identities with search results, prioritizing search when available
  const allIdentities = useMemo(() => {
    if (searchValue.trim() && searchResults.length > 0) {
      // When searching, use search results
      return searchResults;
    }
    if (searchValue.trim() && searchResults.length === 0 && !isSearching) {
      // No search results found, filter from loaded identities as fallback
      const searchTerm = searchValue.toLowerCase().trim();
      return identities.filter((identity) =>
        identity.externalId.toLowerCase().includes(searchTerm),
      );
    }
    // No search query, use all loaded identities
    return identities;
  }, [identities, searchResults, searchValue, isSearching]);

  const handleCreateIdentity = () => {
    if (searchValue.trim()) {
      createIdentity.mutate({
        externalId: searchValue.trim(),
        meta: null,
      });
    }
  };

  const exactMatch = allIdentities.some(
    (id) => id.externalId.toLowerCase() === searchValue.toLowerCase().trim(),
  );

  const filteredIdentities = searchValue.trim()
    ? allIdentities.filter((identity) =>
        identity.externalId.toLowerCase().includes(searchValue.toLowerCase().trim()),
      )
    : allIdentities;

  const hasPartialMatches = filteredIdentities.length > 0;

  // Don't show load more when actively searching
  const showLoadMore = !searchValue.trim() && hasNextPage;

  const baseOptions = createIdentityOptions({
    identities: filteredIdentities,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    loadMore,
  });

  const createOption =
    searchValue.trim() && !exactMatch && hasPartialMatches && !isSearching
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
                <TriangleWarning2 size="sm-regular" />
              </div>
              <span className="text-[13px] text-gray-12 ">
                <span className="text-accent-10 font-normal">Create</span> "{searchValue.trim()}"
              </span>
            </div>
          ),
          value: "__create_new__",
          selectedLabel: <></>,
          searchValue: searchValue.trim(),
        }
      : null;

  const options = createOption ? [createOption, ...baseOptions] : baseOptions;

  // Determine if we're in a transitional state
  const isTransitioning = isSearching || isPending;

  return (
    <FormCombobox
      optional
      label="External ID"
      description="ID of the user/workspace in your system for key attribution."
      options={options}
      key={value}
      value={value || ""}
      onChange={(e) => {
        const newValue = e.currentTarget.value;
        startTransition(() => {
          setSearchValue(newValue);
        });
      }}
      onSelect={(val) => {
        if (val === "__create_new__") {
          handleCreateIdentity();
          return;
        }
        const identity = allIdentities.find((id) => id.id === val);
        onChange(identity?.id || null, identity?.externalId || null);
      }}
      placeholder={
        <div className="relative flex w-full text-grayA-8 text-xs items-center py-2">
          <div
            className={cn(
              "absolute inset-0 flex items-center gap-1.5 transition-opacity duration-200 ease-in-out",
              isTransitioning ? "opacity-100" : "opacity-0",
            )}
          >
            <Loading />
            <span>Searching...</span>
          </div>
          <div
            className={cn(
              "flex items-center transition-opacity duration-200 ease-in-out",
              isTransitioning ? "opacity-0" : "opacity-100",
            )}
          >
            Select External ID
          </div>
        </div>
      }
      searchPlaceholder="Search External ID..."
      emptyMessage={
        searchValue.trim() && !exactMatch ? (
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
                  <TriangleWarning2 size="sm-regular" />
                </div>
                <div className="font-medium text-[13px] leading-7 text-gray-12">
                  External ID not found
                </div>
              </div>
            </div>
            <div className="w-full">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
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
                disabled={!searchValue.trim() || createIdentity.isLoading || disabled}
              >
                Create
              </Button>
            </div>
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
      disabled={disabled}
    />
  );
};
