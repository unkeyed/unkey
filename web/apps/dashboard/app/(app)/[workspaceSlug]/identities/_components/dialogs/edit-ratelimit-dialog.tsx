"use client";

import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import { useEditIdentityRatelimits } from "@/hooks/use-edit-ratelimits";
import type { RatelimitFormValues } from "@/lib/schemas/ratelimit";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer } from "@unkey/ui";
import { type FC, useEffect, useId, useRef, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import type { z } from "zod";
import { IdentityInfo } from "./identity-info";

type Identity = z.infer<typeof IdentityResponseSchema>;

interface EditRatelimitDialogProps {
  identity: Identity;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const getIdentityRatelimitsDefaults = (identity: Identity): RatelimitFormValues => {
  const hasRatelimits = identity.ratelimits && identity.ratelimits.length > 0;
  const defaultRatelimits = hasRatelimits
    ? identity.ratelimits.map((rl) => ({
        id: rl.id,
        name: rl.name,
        limit: rl.limit,
        refillInterval: rl.duration,
        autoApply: rl.autoApply,
      }))
    : [
        {
          name: "Default",
          limit: 10,
          refillInterval: 1000,
          autoApply: false,
        },
      ];

  return {
    ratelimit: {
      enabled: hasRatelimits ? (true as const) : (false as const),
      data: defaultRatelimits,
    },
  } as RatelimitFormValues;
};

export const EditRatelimitDialog: FC<EditRatelimitDialogProps> = ({
  identity,
  open,
  onOpenChange,
}) => {
  const formId = useId();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const identityRef = useRef(identity);
  identityRef.current = identity;

  const methods = useForm<RatelimitFormValues>({
    resolver: zodResolver(ratelimitSchema) as DiscriminatedUnionResolver<typeof ratelimitSchema>,
    defaultValues: getIdentityRatelimitsDefaults(identity),
  });

  // Reset form only when the dialog transitions from closed to open, so
  // refetches of the underlying identity (which can change `identity`'s
  // reference) don't clobber in-progress edits like a newly added ratelimit.
  useEffect(() => {
    if (open) {
      methods.reset(getIdentityRatelimitsDefaults(identityRef.current));
    }
  }, [open, methods]);

  const updateRatelimit = useEditIdentityRatelimits(() => {
    onOpenChange(false);
  });

  const onSubmit = methods.handleSubmit(async (data) => {
    setIsSubmitting(true);
    try {
      await updateRatelimit.mutateAsync({
        identityId: identity.id,
        ratelimit: data.ratelimit,
      });
    } catch {
      // `useEditRatelimits` already shows a toast, but we still need to
      // prevent unhandled rejection noise in the console.
    } finally {
      setIsSubmitting(false);
    }
  });

  return (
    <FormProvider {...methods}>
      <form id={formId} onSubmit={onSubmit}>
        <DialogContainer
          isOpen={open}
          onOpenChange={onOpenChange}
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
                disabled={isSubmitting}
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
