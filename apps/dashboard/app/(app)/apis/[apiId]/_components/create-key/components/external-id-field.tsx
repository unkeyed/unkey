import { useCreateIdentity } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-create-identity";
import { useFetchIdentities } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities";
import { createIdentityOptions } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities/create-identity-options";
import { FormCombobox } from "@/components/ui/form-combobox";
import { TriangleWarning2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";

type ExternalIdFieldProps = {
  value: string | null;
  onChange: (id: string | null) => void;
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
  const { identities, isFetchingNextPage, hasNextPage, loadMore } = useFetchIdentities();

  const createIdentity = useCreateIdentity((data) => {
    onChange(data.identityId);
  });

  const handleCreateIdentity = () => {
    if (searchValue.trim()) {
      createIdentity.mutate({
        externalId: searchValue.trim(),
        meta: null,
      });
    }
  };

  const exactMatch = identities.some(
    (id) => id.externalId.toLowerCase() === searchValue.toLowerCase().trim(),
  );

  const filteredIdentities = searchValue.trim()
    ? identities.filter((identity) =>
        identity.externalId.toLowerCase().includes(searchValue.toLowerCase().trim()),
      )
    : identities;

  const hasPartialMatches = filteredIdentities.length > 0;

  const baseOptions = createIdentityOptions({
    identities: filteredIdentities,
    hasNextPage,
    isFetchingNextPage,
    loadMore,
  });

  const createOption =
    searchValue.trim() && !exactMatch && hasPartialMatches
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

  return (
    <FormCombobox
      optional
      label="External ID"
      description="ID of the user/workspace in your system for key attribution."
      options={options}
      key={value}
      value={value || ""}
      onChange={(e) => setSearchValue(e.currentTarget.value)}
      onSelect={(val) => {
        if (val === "__create_new__") {
          handleCreateIdentity();
          return;
        }
        const identity = identities.find((id) => id.id === val);
        onChange(identity?.id || null);
      }}
      placeholder={
        <div className="flex w-full text-grayA-8 text-xs gap-1.5 items-center py-2">
          Select External ID
        </div>
      }
      searchPlaceholder="Search External ID..."
      emptyMessage={
        searchValue.trim() && !exactMatch ? (
          <div className="p-0 max-w-[460px]">
            <div className="px-3 py-3 w-full">
              <div className="flex gap-2 items-center justify-start">
                <div
                  className={cn(
                    "flex items-center rounded size-5 justify-center",
                    "bg-warningA-4",
                    "text-warning-11",
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
                className="rounded-lg w-full"
                onClick={handleCreateIdentity}
                loading={createIdentity.isLoading}
                disabled={!searchValue.trim() || createIdentity.isLoading || disabled}
              >
                Create
              </Button>
            </div>
          </div>
        ) : (
          <div className="px-3 py-3 text-gray-10 text-[13px]">No results found</div>
        )
      }
      variant="default"
      error={error}
      disabled={disabled}
    />
  );
};
