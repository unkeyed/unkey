"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useCreateIdentityMutation } from "@/lib/identities-query";
import { routes } from "@/lib/navigation/routes";
import { parseIdentityMetadata } from "@/lib/schemas/metadata";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ConflictErrorResponse } from "@unkey/api/models/errors";
import { Plus } from "@unkey/icons";
import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { SECTIONS } from "./create-identity.constants";
import { type FormValues, formSchema, getDefaultValues } from "./create-identity.schema";

export function CreateIdentityDialog() {
  const [open, setOpen] = useState(false);
  const [isNavigating, startNavigation] = useTransition();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

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
    try {
      const createdIdentity = await createIdentity.mutateAsync({
        externalId: data.externalId,
        meta,
        ratelimits,
      });
      reset(getDefaultValues());
      startNavigation(() => {
        router.push(
          routes.identities.detail({
            workspaceSlug: workspace.slug,
            identityId: createdIdentity.identityId,
          }),
        );
      });
    } catch (error) {
      if (error instanceof ConflictErrorResponse) {
        setError(
          "externalId",
          {
            message: "An identity with this external ID already exists",
          },
          { shouldFocus: true },
        );
      }
    }
  };

  const isBusy = createIdentity.isLoading || isNavigating;
  const submissionError =
    createIdentity.isError && !(createIdentity.error instanceof ConflictErrorResponse)
      ? createIdentity.error
      : undefined;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && isBusy) {
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
      <Button size="md" variant="primary" onClick={() => setOpen(true)}>
        <Plus iconSize="sm-regular" />
        Create identity
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
              <div className="flex w-full flex-col gap-3">
                {submissionError ? (
                  <Alert variant="alert">
                    <AlertTitle>Couldn&apos;t Create Identity</AlertTitle>
                    <AlertDescription>
                      {getErrorMessage(submissionError)} Try again.
                    </AlertDescription>
                  </Alert>
                ) : null}
                <div className="flex justify-center items-center w-full">
                  <div className="flex flex-col items-center justify-center w-2/3 gap-2">
                    <Button
                      type="submit"
                      form="create-identity-form"
                      variant="primary"
                      size="xlg"
                      className="w-full rounded-lg"
                      disabled={!isValid || isBusy}
                      loading={isBusy}
                    >
                      Create Identity
                    </Button>
                    <div className="text-gray-9 text-xs">
                      Create an identity to group keys and manage permissions
                    </div>
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
