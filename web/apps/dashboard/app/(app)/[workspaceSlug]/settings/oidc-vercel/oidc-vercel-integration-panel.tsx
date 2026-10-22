"use client";

import { buildUrns } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/urn";
import { grantsToPolicies } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/urn-parse";
import { PolicyListEditor } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/policy-list";
import { rootKeySchema } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/schema";
import { Switch } from "@/components/ui/switch";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { oidcVercelInputSchema } from "@/lib/trpc/routers/oidc-vercel/validation";
import { zodResolver } from "@hookform/resolvers/zod";
import type { inferRouterOutputs } from "@trpc/server";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  CopyButton,
  FormInput,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { getVercelOIDCSnippet } from "./vercel-oidc-snippet";

type Integration = inferRouterOutputs<Router>["oidcVercel"]["list"][number];
const formSchema = oidcVercelInputSchema.omit({ permissions: true }).extend({
  enabled: z.boolean(),
  policies: z.array(rootKeySchema.shape.policies.element),
});
type FormValues = z.infer<typeof formSchema>;

const emptyValues: FormValues = {
  name: "",
  vercelTeamSlug: "",
  vercelProjectName: "",
  vercelEnvironment: "production",
  policies: [],
  enabled: true,
};

function OIDCVercelIntegrationCreated({
  audience,
  onClose,
}: {
  audience: string;
  onClose: () => void;
}) {
  const snippet = getVercelOIDCSnippet(audience);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-4">
        <div className="flex flex-col gap-1">
          <h3 className="text-sm font-medium text-gray-12">Integration created</h3>
          <p className="text-sm text-gray-10">
            Add the audience to your Vercel application before making requests to the Unkey API.
          </p>
        </div>
        <FormInput
          label="Audience"
          description="Pass this value to getVercelOidcToken."
          value={audience}
          copyValue={audience}
          readOnly
          onFocus={(event) => event.currentTarget.select()}
        />
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-1">
            <h3 className="text-sm font-medium text-gray-12">TypeScript example</h3>
            <p className="text-xs text-gray-10">
              Add @vercel/oidc to your Vercel project, then use this route handler.
            </p>
          </div>
          <div className="flex items-start gap-4 rounded-lg border border-grayA-4 bg-grayA-2 p-4">
            <pre className="min-w-0 flex-1 overflow-x-auto text-left font-mono text-xs leading-relaxed text-gray-12">
              <code>{snippet}</code>
            </pre>
            <CopyButton value={snippet} toastMessage="TypeScript example copied" />
          </div>
        </div>
      </div>
      <SlidePanelFooter className="flex justify-end">
        <Button type="button" variant="primary" onClick={onClose}>
          Close
        </Button>
      </SlidePanelFooter>
    </div>
  );
}

export function OIDCVercelIntegrationPanel({
  isOpen,
  onClose,
  integration,
}: {
  isOpen: boolean;
  onClose: () => void;
  integration?: Integration;
}) {
  const workspace = useWorkspaceNavigation();
  const grants = grantsToPolicies(
    workspace.id,
    integration?.permissions.map(({ slug }) => slug) ?? [],
  );
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [createdAudience, setCreatedAudience] = useState<string>();
  const utils = trpc.useUtils();
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    values: integration
      ? {
          name: integration.name,
          vercelTeamSlug: integration.vercelTeamSlug,
          vercelProjectName: integration.vercelProjectName,
          vercelEnvironment: integration.vercelEnvironment,
          policies: grants.policies,
          enabled: integration.enabled,
        }
      : emptyValues,
    mode: "onChange",
  });

  const create = trpc.oidcVercel.create.useMutation({
    onSuccess: async (created) => {
      setCreatedAudience(created.audience);
      await utils.oidcVercel.list.invalidate();
    },
    onError: (error) => toast.error(error.message),
  });
  const update = trpc.oidcVercel.update.useMutation({
    onSuccess: async () => {
      await utils.oidcVercel.list.invalidate();
      toast.success("Changes saved");
      onClose();
    },
    onError: (error) => toast.error(error.message),
  });
  const remove = trpc.oidcVercel.delete.useMutation({
    onSuccess: async () => {
      await utils.oidcVercel.list.invalidate();
      toast.success("Vercel OIDC integration deleted");
      setConfirmDelete(false);
      onClose();
    },
    onError: (error) => toast.error(error.message),
  });

  const pending = create.isLoading || update.isLoading || remove.isLoading;
  const submit = form.handleSubmit(({ enabled, policies, ...values }) => {
    const permissions = [...new Set([...buildUrns(workspace.id, policies), ...grants.unmapped])];
    if (integration) {
      update.mutate({ id: integration.id, enabled, ...values, permissions });
      return;
    }
    create.mutate({ ...values, permissions });
  });
  const reset = () => {
    form.reset(emptyValues);
    setCreatedAudience(undefined);
  };

  return (
    <>
      <SlidePanel isOpen={isOpen} onClose={onClose} onExitComplete={reset} widthClassName="w-192">
        <SlidePanelHeader className="items-center">
          <SlidePanelTitle>
            {createdAudience
              ? "Vercel OIDC Integration Created"
              : integration
                ? "Vercel OIDC Settings"
                : "New Vercel OIDC Integration"}
          </SlidePanelTitle>
          <SlidePanelCloseButton />
        </SlidePanelHeader>
        <SlidePanelContent className="flex flex-col">
          {createdAudience ? (
            <OIDCVercelIntegrationCreated audience={createdAudience} onClose={onClose} />
          ) : (
            <FormProvider {...form}>
              <form onSubmit={submit} className="flex min-h-0 flex-1 flex-col">
                <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-4">
                  {integration && (
                    <FormInput
                      label="Audience"
                      description="Pass this value to getVercelOidcToken."
                      value={integration.audience}
                      copyValue={integration.audience}
                      readOnly
                      onFocus={(event) => event.currentTarget.select()}
                    />
                  )}
                  <FormInput
                    requirement="required"
                    label="Name"
                    description="Shown in the integration list."
                    placeholder="Production API"
                    error={form.formState.errors.name?.message}
                    {...form.register("name")}
                  />
                  <FormInput
                    requirement="required"
                    label="Vercel team slug"
                    description="The owner value in the Vercel OIDC subject."
                    placeholder="acme"
                    error={form.formState.errors.vercelTeamSlug?.message}
                    {...form.register("vercelTeamSlug")}
                  />
                  <FormInput
                    requirement="required"
                    label="Vercel project name"
                    description="The project value in the Vercel OIDC subject."
                    placeholder="storefront"
                    error={form.formState.errors.vercelProjectName?.message}
                    {...form.register("vercelProjectName")}
                  />
                  <FormInput
                    requirement="required"
                    label="Vercel environment"
                    description="Use development, preview, production, or a Custom Environment slug."
                    placeholder="production"
                    error={form.formState.errors.vercelEnvironment?.message}
                    {...form.register("vercelEnvironment")}
                  />
                  <fieldset disabled={pending} className="flex flex-col gap-3">
                    <legend className="mb-3 text-sm text-gray-10">Permissions</legend>
                    {isOpen && (
                      <Controller
                        control={form.control}
                        name="policies"
                        render={({ field }) => (
                          <PolicyListEditor
                            key={integration?.id ?? "create"}
                            policies={field.value}
                            onChange={field.onChange}
                            errors={form.formState.errors.policies}
                          />
                        )}
                      />
                    )}
                    {grants.unmapped.length > 0 && (
                      <div className="flex flex-col gap-2 rounded-lg border p-4">
                        <p className="text-xs text-gray-10">
                          These existing grants cannot be edited here. They will be kept when you
                          save.
                        </p>
                        {grants.unmapped.map((grant) => (
                          <code key={grant} className="break-all text-xs text-gray-11">
                            {grant}
                          </code>
                        ))}
                      </div>
                    )}
                  </fieldset>
                  {integration && (
                    <Controller
                      control={form.control}
                      name="enabled"
                      render={({ field }) => (
                        <div className="flex items-start justify-between gap-4 rounded-lg border border-grayA-4 p-4">
                          <div className="flex flex-col gap-1">
                            <span className="text-sm font-medium text-gray-12">Enabled</span>
                            <span className="text-xs text-gray-10">
                              Disabled integrations reject all Vercel OIDC tokens.
                            </span>
                          </div>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                            disabled={pending}
                          />
                        </div>
                      )}
                    />
                  )}
                </div>
                <SlidePanelFooter className="flex items-center justify-between gap-3">
                  {integration ? (
                    <Button
                      type="button"
                      variant="ghost"
                      className="text-error-11"
                      onClick={() => setConfirmDelete(true)}
                      disabled={pending}
                    >
                      Delete Integration
                    </Button>
                  ) : (
                    <span />
                  )}
                  <Button
                    type="submit"
                    variant="primary"
                    loading={pending}
                    disabled={!form.formState.isValid}
                  >
                    {integration ? "Save Changes" : "Create Integration"}
                  </Button>
                </SlidePanelFooter>
              </form>
            </FormProvider>
          )}
        </SlidePanelContent>
      </SlidePanel>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Vercel OIDC integration?</AlertDialogTitle>
            <AlertDialogDescription>
              Requests that use this integration will lose access. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep Integration</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => integration && remove.mutate({ id: integration.id })}
            >
              Delete Integration
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
