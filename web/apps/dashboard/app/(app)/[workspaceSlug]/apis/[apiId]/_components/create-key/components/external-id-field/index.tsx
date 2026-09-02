import type { Identity } from "@unkey/api/models/components";
import { BadRequestErrorResponse, ConflictErrorResponse } from "@unkey/api/models/errors";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { IconTriangleWarningOutline12 } from "nucleo-ui-outline-12";
import { useMemo, useState } from "react";
import { FormCombobox } from "@/components/ui/form-combobox";
import { useCreateIdentityMutation, useIdentities } from "@/lib/identities-query";
import { identityExternalIdSchema } from "@/lib/schemas/identity";
import { getErrorMessage } from "@/lib/unkey-client";
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
  const [selectedIdentity, setSelectedIdentity] = useState<Identity>();
  const [nextPageError, setNextPageError] = useState<unknown>();

  const trimmedSearchValue = searchValue.trim();
  const externalIdValidation = identityExternalIdSchema.safeParse(trimmedSearchValue);
  const externalIdError = externalIdValidation.success
    ? undefined
    : externalIdValidation.error.issues.at(0)?.message;
  const {
    identities,
    isFetching,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
    isLoading,
    isError: isIdentitiesError,
    error: identitiesError,
    refetch: refetchIdentities,
  } = useIdentities({ search: trimmedSearchValue || undefined });
  const isSearching = trimmedSearchValue.length > 0 && isFetching;
  const loadMore = async () => {
    try {
      await fetchNextPage({ throwOnError: true });
      setNextPageError(undefined);
    } catch (error) {
      setNextPageError(error);
    }
  };
  const retryIdentities = () => {
    refetchIdentities().catch((error: unknown) => {
      console.error("Failed to retry identities query", error);
    });
  };

  const createIdentity = useCreateIdentityMutation();

  // Ensure current identity is always available in the options
  const allIdentitiesWithCurrent = useMemo(() => {
    if (!value || identities.some((identity) => identity.id === value)) {
      return identities;
    }

    const current =
      currentIdentity?.id === value
        ? {
            id: currentIdentity.id,
            externalId: currentIdentity.externalId,
            meta: currentIdentity.meta,
          }
        : selectedIdentity?.id === value
          ? selectedIdentity
          : undefined;
    if (!current) {
      return identities;
    }

    return [current, ...identities];
  }, [identities, currentIdentity, selectedIdentity, value]);

  const selectedExternalId = useMemo(() => {
    if (!value) {
      return undefined;
    }
    return allIdentitiesWithCurrent.find((id) => id.id === value)?.externalId;
  }, [allIdentitiesWithCurrent, value]);

  const handleCreateIdentity = async () => {
    if (!externalIdValidation.success) {
      return;
    }

    const externalId = externalIdValidation.data;

    // Ask the server for this ID once more before creating: the "create"
    // offer was rendered from whatever results were on screen, which may be
    // stale. If the ID already exists, select it instead of duplicating it.
    const fresh = await refetchIdentities();
    const existing = fresh.data?.pages
      .flatMap((page) => page.identities)
      .find((identity) => identity.externalId.toLowerCase() === externalId.toLowerCase());
    if (existing) {
      setSelectedIdentity(existing);
      setSearchValue("");
      onChange(existing.id, existing.externalId);
      return;
    }

    try {
      const data = await createIdentity.mutateAsync({ externalId });
      setSelectedIdentity({ id: data.identityId, externalId: data.externalId });
      setSearchValue("");
      onChange(data.identityId, data.externalId);
    } catch {
      // The mutation error is rendered beneath the combobox.
    }
  };

  const createIdentityError = createIdentity.isError
    ? createIdentity.error instanceof ConflictErrorResponse
      ? "An identity with this external ID already exists in your workspace."
      : createIdentity.error instanceof BadRequestErrorResponse
        ? getErrorMessage(createIdentity.error, "Check the external ID and try again.")
        : `${getErrorMessage(createIdentity.error)} Try again.`
    : undefined;

  const exactMatch = allIdentitiesWithCurrent.some(
    (id) => id.externalId.toLowerCase() === trimmedSearchValue.toLowerCase(),
  );

  const hasPartialMatches = allIdentitiesWithCurrent.length > 0;

  // Don't show load more when actively searching
  const showLoadMore = !trimmedSearchValue && hasNextPage;
  const optionQueryError = nextPageError
    ? {
        message: getErrorMessage(nextPageError, "We couldn't load more identities."),
        retry: loadMore,
      }
    : isIdentitiesError && allIdentitiesWithCurrent.length > 0
      ? {
          message: getErrorMessage(identitiesError, "We couldn't refresh identities."),
          retry: retryIdentities,
        }
      : undefined;

  const baseOptions = createIdentityOptions({
    identities: allIdentitiesWithCurrent,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    queryError: optionQueryError?.message,
  });

  const createOption =
    externalIdValidation.success && !exactMatch && hasPartialMatches
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
                <IconTriangleWarningOutline12 />
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

  const isComboboxLoading = trimmedSearchValue ? isFetching : isLoading;
  const initialQueryError = isIdentitiesError && allIdentitiesWithCurrent.length === 0;

  return (
    <FormCombobox
      requirement="optional"
      label="External ID"
      description={
        <>
          ID of the user/workspace in your system for key attribution.
          {optionQueryError ? (
            <span role="alert" className="sr-only">
              {optionQueryError.message} Select the retry option to try again.
            </span>
          ) : null}
        </>
      }
      options={options}
      key={value}
      value={value || ""}
      onChange={(e) => {
        if (!createIdentity.isLoading) {
          createIdentity.reset();
        }
        setNextPageError(undefined);
        setSearchValue(e.currentTarget.value);
      }}
      onSelect={async (val) => {
        if (val === "__load_more__") {
          if (!isFetchingNextPage) {
            loadMore();
          }
          return;
        }
        if (val === "__retry_identities__") {
          optionQueryError?.retry();
          return;
        }
        if (val === "__create_new__") {
          await handleCreateIdentity();
          return;
        }
        const identity = allIdentitiesWithCurrent.find((id) => id.id === val);
        createIdentity.reset();
        setSelectedIdentity(identity);
        setSearchValue("");
        onChange(identity?.id || null, identity?.externalId || null);
      }}
      placeholder={
        <div className="flex w-full text-grayA-8 text-xs items-center py-2">Select External ID</div>
      }
      searchPlaceholder="Search External ID..."
      emptyMessage={
        initialQueryError ? (
          <div role="alert" className="flex flex-col gap-3 px-4 py-4 text-left">
            <div className="text-error-11 text-[13px] leading-5">
              {getErrorMessage(
                identitiesError,
                trimmedSearchValue
                  ? "We couldn't search identities."
                  : "We couldn't load identities.",
              )}
            </div>
            <Button type="button" variant="outline" size="md" onClick={retryIdentities}>
              Retry
            </Button>
          </div>
        ) : trimmedSearchValue && !exactMatch && !isComboboxLoading ? (
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
                  <IconTriangleWarningOutline12 />
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
      error={createIdentityError ?? error}
      disabled={disabled || isLoading || createIdentity.isLoading}
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
