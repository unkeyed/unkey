"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronLeft } from "@unkey/icons";
import {
  Button,
  FormInput,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
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

  usePreventLeave(formState.isDirty && created === null);

  const submit = form.handleSubmit(
    (values) => {
      if (values.policies.length === 0 || !values.policies.every(isPolicyComplete)) {
        setValidated(true);
        return;
      }
      setReviewing(true);
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
      <PageContainer>
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>{reviewing ? "Review key" : "New Root Key"}</PageHeaderTitle>
          </PageHeaderContent>
        </PageHeader>
        <PageBody>
          <form onSubmit={submit} className="flex flex-col gap-6">
            {reviewing ? (
              <ReviewStage name={values.name.trim()} policies={values.policies} />
            ) : (
              <>
                <Controller
                  control={control}
                  name="name"
                  render={({ field, fieldState }) => (
                    <FormInput
                      label="Name"
                      requirement="required"
                      placeholder="e.g. CI deploy key"
                      value={field.value}
                      onChange={field.onChange}
                      error={fieldState.error?.message}
                    />
                  )}
                />

                <div className="flex flex-col gap-2">
                  <span className="text-[13px] text-gray-11">Permissions</span>
                  <PolicyList showErrors={validated} />
                </div>
              </>
            )}

            <div className="flex items-center gap-3 border-t border-grayA-4 pt-5">
              {reviewing ? (
                <>
                  <Button
                    type="button"
                    variant="ghost"
                    size="md"
                    onClick={() => setReviewing(false)}
                  >
                    <ChevronLeft iconSize="sm-regular" />
                    Back to edit
                  </Button>
                  <Button
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
                <>
                  <Button
                    variant="ghost"
                    size="md"
                    className="ml-auto"
                    render={
                      <Link href={routes.settings.rootKeys({ workspaceSlug: workspace.slug })} />
                    }
                  >
                    Cancel
                  </Button>
                  <Button type="submit" variant="primary" size="md">
                    Review key
                  </Button>
                </>
              )}
            </div>
          </form>
        </PageBody>
      </PageContainer>
      {created ? (
        <SuccessDialog
          keyId={created.keyId}
          secret={created.secret}
          onDone={() => router.push(routes.settings.rootKeys({ workspaceSlug: workspace.slug }))}
        />
      ) : null}
    </FormProvider>
  );
}
