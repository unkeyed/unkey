"use client";

import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import { metadataSchema } from "@/lib/schemas/metadata";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import { trpc } from "@/lib/trpc/client";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { type FC, useCallback, useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import type { z } from "zod";
import { IdentityInfo } from "../dialogs/identity-info";

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
    resolver: zodResolver(metadataSchema) as DiscriminatedUnionResolver<typeof metadataSchema>,
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
          <IdentityInfo identity={identity} />
          <div className="py-1 my-2">
            <div className="h-px bg-grayA-3 w-full" />
          </div>
          <MetadataSetup entityType="identity" />
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
