"use client";

import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { useUpdateIdentityMutation } from "@/lib/identities-query";
import type { RatelimitFormValues } from "@/lib/schemas/ratelimit";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Identity } from "@unkey/api/models/components";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { type FC, useEffect, useId } from "react";
import { FormProvider } from "react-hook-form";
import { IdentityInfo } from "./identity-info";

type EditRatelimitDialogProps = { identity: Identity } & ActionComponentProps;

const EDIT_RATELIMITS_FORM_STORAGE_KEY = "unkey_edit_identity_ratelimits_form_state";

const getIdentityRatelimitsDefaults = (identity: Identity) => {
  const ratelimits = identity.ratelimits ?? [];
  const hasRatelimits = ratelimits.length > 0;

  return {
    ratelimit: hasRatelimits
      ? ({
          enabled: true as const,
          data: ratelimits.map((rl) => ({
            id: rl.id,
            name: rl.name,
            limit: rl.limit,
            refillInterval: rl.duration,
            autoApply: rl.autoApply,
          })),
        } as const)
      : ({ enabled: false as const } as const),
  };
};

export const EditRatelimitDialog: FC<EditRatelimitDialogProps> = ({
  identity,
  isOpen,
  onClose,
}) => {
  const formId = useId();
  const updateIdentity = useUpdateIdentityMutation();

  const methods = usePersistedForm<RatelimitFormValues>(
    `${EDIT_RATELIMITS_FORM_STORAGE_KEY}_${identity.id}`,
    {
      resolver: zodResolver(ratelimitSchema) as DiscriminatedUnionResolver<typeof ratelimitSchema>,
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getIdentityRatelimitsDefaults(identity),
    },
    "memory",
  );

  const {
    handleSubmit,
    formState: { isSubmitting, isValid },
    loadSavedValues,
    saveCurrentValues,
    clearPersistedData,
    reset,
  } = methods;

  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const onSubmit = async (data: RatelimitFormValues) => {
    try {
      const mutation = updateIdentity.mutateAsync({
        identity: identity.id,
        ratelimits: data.ratelimit.enabled
          ? data.ratelimit.data.map((rule) => ({
              name: rule.name,
              limit: rule.limit,
              duration: rule.refillInterval,
              autoApply: rule.autoApply,
            }))
          : [],
      });
      toast.promise(mutation, {
        loading: "Updating rate limits...",
        success: {
          message: "Identity Ratelimits Updated",
          description: `Rate limits for identity ${identity.id} have been updated`,
          duration: 5000,
        },
        error: (error) => ({
          message: "Failed to Update Identity Limits",
          description: getErrorMessage(error),
        }),
      });
      const updatedIdentity = await mutation;
      reset(getIdentityRatelimitsDefaults(updatedIdentity));
      clearPersistedData();
      onClose();
    } catch {
      // toast.promise reports the API error.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id={formId} onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          onOpenChange={(o) => {
            if (!o && !isSubmitting) {
              saveCurrentValues();
              onClose();
            }
          }}
          title="Edit ratelimit"
          subTitle="Control how often this identity can be used"
          className="flex flex-col"
          contentClassName="flex flex-col flex-1 min-h-0"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form={formId}
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                loading={isSubmitting}
              >
                Update ratelimit
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          {/* Scrollable body container */}
          <div className="flex-1 overflow-y-auto min-h-0 scrollbar-hide gap-4 flex flex-col">
            <IdentityInfo identity={identity} />
            <div className="py-1 my-2">
              <div className="h-px bg-grayA-3 w-full" />
            </div>
            <div className="[&>*:first-child]:p-0">
              <RatelimitSetup entityType="identity" />
            </div>
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
