import { useCreateIdentity } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-create-identity";
import { useFetchIdentities } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities";
import { createIdentityOptions } from "@/app/(app)/apis/[apiId]/_components/create-key/hooks/use-fetch-identities/create-identity-options";
import { FormCombobox } from "@/components/ui/form-combobox";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
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
  const identityOptions = createIdentityOptions({
    identities,
    hasNextPage,
    isFetchingNextPage,
    loadMore,
  });

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

  return (
    <FormCombobox
      optional
      label="External ID"
      description="ID of the user/workspace in your system for key attribution."
      options={identityOptions}
      // This will force close popover whenever new item added or selected.
      key={value}
      value={value || ""}
      onChange={(e) => setSearchValue(e.currentTarget.value)}
      onSelect={(val) => {
        const identity = identities.find((id) => id.id === val);
        onChange(identity?.id || null);
      }}
      placeholder={
        <div className="flex w-full text-grayA-8 text-xs gap-1.5 items-center py-2">
          Select external ID
        </div>
      }
      searchPlaceholder="Search external ID..."
      emptyMessage={
        <div className="flex flex-col gap-4 items-center justify-center py-2">
          <span className="text-gray-9 text-sm">No external ID found.</span>
          <Button
            variant="outline"
            size="md"
            className="w-fit rounded-lg"
            onClick={handleCreateIdentity}
            loading={createIdentity.isLoading}
            disabled={!searchValue.trim() || createIdentity.isLoading || disabled}
          >
            <Plus size="sm-regular" />
            Create new identity
          </Button>
        </div>
      }
      variant="default"
      error={error}
      disabled={disabled}
    />
  );
};
