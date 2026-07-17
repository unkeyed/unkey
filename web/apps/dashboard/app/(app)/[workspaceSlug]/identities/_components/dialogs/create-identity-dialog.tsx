"use client";

import { useCreateIdentityMutation } from "@/lib/identities-query";
import { parseIdentityMetadata } from "@/lib/schemas/metadata";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ConflictErrorResponse } from "@unkey/api/models/errors";
import { Plus } from "@unkey/icons";
import {
  Button,
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { SECTIONS } from "./create-identity.constants";
import { type FormValues, formSchema, getDefaultValues } from "./create-identity.schema";

export function CreateIdentityDialog() {
  const [open, setOpen] = useState(false);

  const methods = useForm<FormValues>({
    resolver: zodResolver(formSchema) as DiscriminatedUnionResolver<typeof formSchema>,
    mode: "onChange",
    defaultValues: getDefaultValues(),
  });

  const {
    handleSubmit,
    setError,
    formState: { isValid },
    reset,
  } = methods;

  const createIdentity = useCreateIdentityMutation();

  const onSubmit = async (data: FormValues) => {
    const meta =
      data.metadata?.enabled && data.metadata.data
        ? parseIdentityMetadata(data.metadata.data)
        : undefined;
    const ratelimits =
      data.ratelimit?.enabled && data.ratelimit.data
        ? data.ratelimit.data.map((ratelimit) => ({
            name: ratelimit.name,
            limit: ratelimit.limit,
            duration: ratelimit.refillInterval,
            autoApply: ratelimit.autoApply,
          }))
        : undefined;
    const mutation = createIdentity.mutateAsync({
      externalId: data.externalId,
      meta,
      ratelimits,
    });
    toast.promise(mutation, {
      loading: "Creating identity...",
      success: (createdIdentity) => ({
        message: "Identity created successfully",
        description: `Identity "${createdIdentity.externalId}" has been created.`,
      }),
      error: (error) => ({
        message:
          error instanceof ConflictErrorResponse
            ? "Identity already exists"
            : "Failed to create identity",
        description:
          error instanceof ConflictErrorResponse
            ? "An identity with this external ID already exists."
            : getErrorMessage(error),
      }),
    });

    try {
      await mutation;
      setOpen(false);
      reset(getDefaultValues());
    } catch (error) {
      if (error instanceof ConflictErrorResponse) {
        setError("externalId", {
          message: "An identity with this external ID already exists",
        });
      }
    }
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && createIdentity.isLoading) {
      return;
    }
    setOpen(nextOpen);
    if (!nextOpen) {
      reset(getDefaultValues());
      createIdentity.reset();
    }
  };

  return (
    <>
      <Button size="md" title="Create Identity" onClick={() => setOpen(true)}>
        <Plus iconSize="md-medium" />
        Create Identity
      </Button>

      <FormProvider {...methods}>
        <form id="create-identity-form" onSubmit={handleSubmit(onSubmit)}>
          <NavigableDialogRoot
            key={open ? "open" : "closed"}
            isOpen={open}
            onOpenChange={handleOpenChange}
            dialogClassName="w-[90%] md:w-[70%] lg:w-[70%] xl:w-[50%] 2xl:w-[45%] max-w-[940px] max-h-[90vh]"
          >
            <NavigableDialogHeader
              title="Create Identity"
              subTitle="Create a new identity to associate with keys and rate limits"
            />
            <NavigableDialogBody>
              <NavigableDialogNav
                items={SECTIONS.map((section) => ({
                  id: section.id,
                  label: section.label,
                  icon: section.icon,
                }))}
                initialSelectedId="general"
              />
              <NavigableDialogContent
                items={SECTIONS.map((section) => ({
                  id: section.id,
                  content: section.content(),
                }))}
              />
            </NavigableDialogBody>
            <NavigableDialogFooter>
              <div className="flex justify-center items-center w-full">
                <div className="flex flex-col items-center justify-center w-2/3 gap-2">
                  <Button
                    type="submit"
                    form="create-identity-form"
                    variant="primary"
                    size="xlg"
                    className="w-full rounded-lg"
                    disabled={!isValid || createIdentity.isLoading}
                    loading={createIdentity.isLoading}
                  >
                    Create Identity
                  </Button>
                  <div className="text-gray-9 text-xs">
                    Create an identity to group keys and manage permissions
                  </div>
                </div>
              </div>
            </NavigableDialogFooter>
          </NavigableDialogRoot>
        </form>
      </FormProvider>
    </>
  );
}
