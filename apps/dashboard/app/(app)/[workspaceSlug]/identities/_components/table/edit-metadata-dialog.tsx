"use client";

import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import { metadataSchema } from "@/lib/schemas/metadata";
import { trpc } from "@/lib/trpc/client";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Fingerprint } from "@unkey/icons";
import { Button, DialogContainer, InfoTooltip, toast } from "@unkey/ui";
import { type FC, useCallback, useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

interface EditMetadataDialogProps {
  identity: Identity;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const EditMetadataDialog: FC<EditMetadataDialogProps> = ({
  identity,
  open,
  onOpenChange,
}) => {
  const utils = trpc.useUtils();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const getDefaultValues = useCallback(() => {
    if (identity.meta) {
      return {
        metadata: {
          enabled: true as const,
          data: JSON.stringify(identity.meta, null, 2),
        },
      };
    }
    return {
      metadata: {
        enabled: false as const,
      },
    };
  }, [identity.meta]);

  const methods = useForm<z.infer<typeof metadataSchema>>({
    resolver: zodResolver(metadataSchema),
    defaultValues: getDefaultValues(),
  });

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      methods.reset(getDefaultValues());
    }
  }, [open, getDefaultValues, methods.reset]);

  const updateMetadata = trpc.identity.update.metadata.useMutation({
    onSuccess: () => {
      toast.success("Metadata updated successfully");
      // Invalidate queries to refresh the data
      utils.identity.query.invalidate();
      utils.identity.getById.invalidate();
      onOpenChange(false);
    },
    onError: (error) => {
      toast.error(error.message || "Failed to update metadata");
    },
    onSettled: () => {
      setIsSubmitting(false);
    },
  });

  const onSubmit = methods.handleSubmit(async (data) => {
    setIsSubmitting(true);
    updateMetadata.mutate({
      identityId: identity.id,
      metadata: data.metadata,
    });
  });

  return (
    <FormProvider {...methods}>
      <form id="edit-identity-metadata-form" onSubmit={onSubmit}>
        <DialogContainer
          isOpen={open}
          onOpenChange={onOpenChange}
          title="Edit metadata"
          subTitle="Attach custom data to this identity"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-identity-metadata-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!methods.formState.isValid || isSubmitting}
                loading={isSubmitting}
              >
                Update metadata
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
            <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded">
              <Fingerprint iconSize="sm-regular" />
            </div>
            <div className="flex flex-col gap-1">
              <div className="text-accent-12 text-xs font-mono">{identity.id}</div>
              <InfoTooltip
                variant="inverted"
                content={identity.externalId}
                position={{ side: "bottom", align: "center" }}
                asChild
              >
                <div className="text-accent-9 text-xs max-w-[160px] truncate">
                  {identity.externalId}
                </div>
              </InfoTooltip>
            </div>
          </div>

          <div className="py-1 my-2">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>
          <MetadataSetup entityType="identity" />
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
