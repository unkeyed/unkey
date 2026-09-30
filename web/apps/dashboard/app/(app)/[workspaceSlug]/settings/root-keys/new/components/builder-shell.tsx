"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { ROOT_KEYS_V2_QUERY_KEY } from "@/lib/root-keys-api";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import {
  IconChevronLeftOutline18,
  IconChevronRightOutline18,
  IconCircleInfoOutline18,
} from "@unkey/icons";
import {
  Button,
  FormInput,
  InfoTooltip,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  RequiredTag,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { createRootKeyFromForm } from "../lib/create-root-key";
import { isPolicyComplete } from "../lib/policy";
import { type RootKeyFormValues, rootKeyDefaultValues, rootKeySchema } from "../schema";
import { DebugPanel } from "./debug-panel";
import { PolicyList } from "./policy-list";
import { ReviewStage } from "./review-stage";
import { SuccessDialog } from "./success-dialog";

type CreatedKey = {
  keyId: string;
  secret: string;
};

export function BuilderShell() {
  const workspace = useWorkspaceNavigation();
  const { workspace: currentWorkspace } = useWorkspace();
  const queryClient = useQueryClient();
  const router = useRouter();
  const [reviewing, setReviewing] = useState(false);
  const [validated, setValidated] = useState(false);
  const [created, setCreated] = useState<CreatedKey | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [debug, setDebug] = useState(false);
  const form = useForm<RootKeyFormValues>({
    resolver: zodResolver(rootKeySchema),
    defaultValues: rootKeyDefaultValues,
  });
  const { control, formState } = form;
  const topRef = useRef<HTMLDivElement>(null);

  usePreventLeave(formState.isDirty && created === null);

  const scrollToTop = () => {
    topRef.current?.scrollIntoView({
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
      block: "start",
    });
  };

  const showStage = (review: boolean) => {
    setReviewing(review);
    scrollToTop();
  };

  const submit = form.handleSubmit(
    (values) => {
      if (values.policies.length === 0 || !values.policies.every(isPolicyComplete)) {
        setValidated(true);
        scrollToTop();
        return;
      }
      showStage(true);
    },
    () => setValidated(true),
  );

  const create = async () => {
    if (!currentWorkspace) {
      return;
    }

    setIsCreating(true);
    try {
      const rootKey = await createRootKeyFromForm(currentWorkspace.id, form.getValues());
      await queryClient.invalidateQueries({ queryKey: ROOT_KEYS_V2_QUERY_KEY });
      setCreated(rootKey);
    } catch (error) {
      toast.error("Failed to create root key", {
        description: error instanceof Error ? error.message : "Please try again.",
      });
    } finally {
      setIsCreating(false);
    }
  };

  const values = form.getValues();

  return (
    <FormProvider {...form}>
      <PageContainer ref={topRef}>
        <PageHeader className="max-w-3xl">
          <PageHeaderContent>
            <PageHeaderTitle>{reviewing ? "Review key" : "New root key"}</PageHeaderTitle>
          </PageHeaderContent>
        </PageHeader>
        <PageBody className="max-w-3xl pt-5">
          <form onSubmit={submit} className="flex flex-col gap-5">
            {reviewing ? (
              <ReviewStage name={values.name.trim()} policies={values.policies} />
            ) : (
              <div className="flex flex-col gap-6 rounded-lg border border-grayA-4 bg-white p-5 dark:bg-black shadow-xs">
                <Controller
                  control={control}
                  name="name"
                  render={({ field, fieldState }) => (
                    <FormInput
                      label="Name"
                      requirement="required"
                      placeholder="e.g. CI deploy key"
                      ref={field.ref}
                      value={field.value}
                      onChange={field.onChange}
                      error={fieldState.error?.message}
                    />
                  )}
                />

                <div className="flex flex-col gap-2">
                  <span className="flex h-5 items-center text-sm text-gray-11">
                    Permissions
                    <InfoTooltip
                      content="Select the privileges you'd like this root key to have."
                      position={{ side: "right" }}
                    >
                      <IconCircleInfoOutline18 className="ml-1.5 shrink-0 text-gray-9" />
                    </InfoTooltip>
                    <RequiredTag hasError={validated && values.policies.length === 0} />
                  </span>
                  <PolicyList showErrors={validated} debug={debug} />
                </div>
              </div>
            )}

            <div className="flex items-center gap-3">
              {reviewing ? (
                <>
                  <Button
                    key="back-to-edit"
                    type="button"
                    variant="ghost"
                    size="md"
                    onClick={() => showStage(false)}
                  >
                    <IconChevronLeftOutline18 />
                    Back to edit
                  </Button>
                  <Button
                    key="create-key"
                    type="button"
                    variant="primary"
                    size="md"
                    className="ml-auto"
                    loading={isCreating}
                    disabled={isCreating}
                    onClick={create}
                  >
                    Create key
                  </Button>
                </>
              ) : (
                <Button type="submit" variant="primary" size="md" className="ml-auto">
                  Review key
                  <IconChevronRightOutline18 />
                </Button>
              )}
            </div>
          </form>
        </PageBody>
      </PageContainer>
      <DebugPanel debug={debug} onDebugChange={setDebug} />
      {created ? (
        <SuccessDialog
          secret={created.secret}
          onDone={() => router.push(routes.settings.rootKeys({ workspaceSlug: workspace.slug }))}
        />
      ) : null}
    </FormProvider>
  );
}
