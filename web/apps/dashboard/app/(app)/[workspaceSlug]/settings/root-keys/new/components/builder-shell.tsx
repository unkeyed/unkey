"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronLeft, ChevronRight, CircleInfo } from "@unkey/icons";
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
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { isPolicyComplete } from "../lib/policy";
import { type RootKeyFormValues, rootKeyDefaultValues, rootKeySchema } from "../schema";
import { PolicyList } from "./policy-list";
import { ReviewStage } from "./review-stage";
import { SuccessDialog } from "./success-dialog";

type CreatedKey = {
  keyId: string;
  secret: string;
};

function stubCreateRootKey(): CreatedKey {
  const suffix = Math.random().toString(36).slice(2, 10);
  return { keyId: `key_${suffix}`, secret: `unkey_root_${suffix}${suffix}${suffix}` };
}

export function BuilderShell() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const [reviewing, setReviewing] = useState(false);
  const [validated, setValidated] = useState(false);
  const [created, setCreated] = useState<CreatedKey | null>(null);
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

  const create = () => {
    // TODO: call the create-root-key mutation once the backend lane lands it.
    setCreated(stubCreateRootKey());
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
                  <span className="flex h-5 items-center text-[13px] text-gray-11">
                    Permissions
                    <InfoTooltip
                      content="Select the privileges you'd like this root key to have."
                      position={{ side: "right" }}
                    >
                      <CircleInfo iconSize="sm-regular" className="ml-1.5 shrink-0 text-gray-9" />
                    </InfoTooltip>
                    <RequiredTag hasError={validated && values.policies.length === 0} />
                  </span>
                  <PolicyList showErrors={validated} />
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
                    <ChevronLeft />
                    Back to edit
                  </Button>
                  <Button
                    key="create-key"
                    type="button"
                    variant="primary"
                    size="md"
                    className="ml-auto"
                    onClick={create}
                  >
                    Create key
                  </Button>
                </>
              ) : (
                <Button
                  key="review-key"
                  type="submit"
                  variant="primary"
                  size="md"
                  className="ml-auto"
                >
                  Review key
                  <ChevronRight />
                </Button>
              )}
            </div>
          </form>
        </PageBody>
      </PageContainer>
      {created ? (
        <SuccessDialog
          secret={created.secret}
          onDone={() => router.push(routes.settings.rootKeys({ workspaceSlug: workspace.slug }))}
        />
      ) : null}
    </FormProvider>
  );
}
