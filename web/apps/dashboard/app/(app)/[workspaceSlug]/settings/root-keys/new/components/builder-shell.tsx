"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput } from "@unkey/ui";
import Link from "next/link";
import { useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { type RootKeyFormValues, rootKeyDefaultValues, rootKeySchema } from "../schema";
import { PolicyList } from "./policy-list";

export function BuilderShell() {
  const workspace = useWorkspaceNavigation();
  const [validated, setValidated] = useState(false);
  const form = useForm<RootKeyFormValues>({
    resolver: zodResolver(rootKeySchema),
    defaultValues: rootKeyDefaultValues,
  });
  const { control } = form;
  const review = () => setValidated(true);

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(review, review)} className="flex flex-col gap-6">
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

        <div className="flex items-center justify-end gap-3 border-t border-grayA-4 pt-5">
          <Button
            variant="ghost"
            size="md"
            render={<Link href={routes.settings.rootKeys({ workspaceSlug: workspace.slug })} />}
          >
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md">
            Review key
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
