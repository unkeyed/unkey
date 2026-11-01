"use client";

import { EditorCatalogueProvider } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/editor-catalogue";
import { useRootKeyPolicyForm } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/hooks/use-root-key-policy-form";
import { KeyFields } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/key-fields";
import {
  CATALOGUES,
  catalogueRows,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/catalogue";
import { RESOURCE_SCOPES } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/catalogue.types";
import {
  type Policy,
  newPolicy,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/policy";
import {
  TEMPLATES,
  type TemplateId,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/templates";
import { buildUrns } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/urn";
import { PolicyList } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/policy-list";
import { TEMPLATE_ICONS } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/template-gallery";
import { grantLabel } from "@/app/(app)/[workspaceSlug]/settings/root-keys/lib/grant-label";
import { grantableCatalogues, uncoveredPermissions } from "@/lib/cli/grants";
import { IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  FormField,
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { type Ref, useMemo, useState } from "react";
import { FormProvider } from "react-hook-form";

const PRESETS = TEMPLATES.flatMap((template) =>
  template.id === "custom" ? [] : [{ ...template, id: template.id }],
);

type Preset = (typeof PRESETS)[number];

function canGrant(grantable: readonly string[], workspaceId: string, policies: Policy[]): boolean {
  return uncoveredPermissions(grantable, buildUrns(workspaceId, policies)).length === 0;
}

export type Access =
  | { kind: "none" }
  | { kind: "preset"; preset: Preset }
  | { kind: "custom"; policies: Policy[] };

export function accessPolicies(access: Access): Policy[] {
  return match(access)
    .with({ kind: "none" }, () => [])
    .with({ kind: "preset" }, ({ preset }) => preset.materialise())
    .with({ kind: "custom" }, ({ policies }) => policies)
    .exhaustive();
}

type PermissionsFieldProps = {
  access: Access;
  onChange: (access: Access) => void;
  workspaceId: string;
  grantable: readonly string[];
  missing: boolean;
  disabled: boolean;
  triggerRef: Ref<HTMLButtonElement>;
};

export function PermissionsField({
  access,
  onChange,
  workspaceId,
  grantable,
  missing,
  disabled,
  triggerRef,
}: PermissionsFieldProps) {
  const [panel, setPanel] = useState({ open: false, key: 0 });
  const presets = PRESETS.filter((preset) =>
    canGrant(grantable, workspaceId, preset.materialise()),
  );
  const openPanel = () => setPanel((current) => ({ open: true, key: current.key + 1 }));
  const closePanel = () => setPanel((current) => ({ ...current, open: false }));

  return (
    <>
      <FormField
        label="Permissions"
        error={missing ? "Choose permissions to continue." : undefined}
      >
        {({ id, variant, describedBy, invalid }) => (
          <div className="flex items-center gap-2">
            <Select
              value={match(access)
                .with({ kind: "none" }, () => null)
                .with({ kind: "preset" }, ({ preset }) => preset.id)
                .with({ kind: "custom" }, () => "custom")
                .exhaustive()}
              disabled={disabled}
              onValueChange={(value) => {
                if (value === "custom") {
                  openPanel();
                  return;
                }
                const preset = presets.find((candidate) => candidate.id === value);
                if (preset) {
                  onChange({ kind: "preset", preset });
                }
              }}
            >
              <SelectTrigger
                ref={triggerRef}
                id={id}
                variant={variant}
                aria-describedby={describedBy}
                aria-invalid={invalid}
                wrapperClassName="flex-1"
              >
                {match(access)
                  .with({ kind: "none" }, () => (
                    <span className="text-grayA-8">Choose permissions</span>
                  ))
                  .with({ kind: "preset" }, ({ preset }) => (
                    <ChosenLabel id={preset.id} label={preset.title} />
                  ))
                  .with({ kind: "custom" }, () => <ChosenLabel id="custom" label="Custom policy" />)
                  .exhaustive()}
              </SelectTrigger>
              <SelectContent>
                {presets.map((preset) => (
                  <SelectItem key={preset.id} value={preset.id}>
                    <span className="flex items-center gap-3">
                      <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-grayA-3 text-gray-11 [&>svg]:size-3.5">
                        {TEMPLATE_ICONS[preset.id]}
                      </span>
                      <span className="flex flex-col">
                        <span className="text-gray-12">{preset.title}</span>
                        <span className="text-xs text-gray-11">{preset.description}</span>
                      </span>
                    </span>
                  </SelectItem>
                ))}
                {presets.length > 0 ? <SelectSeparator /> : null}
                <SelectItem value="custom">
                  <span className="flex items-center gap-2 text-gray-11">
                    <IconPlusOutline18 className="size-3.5" />
                    {access.kind === "custom" ? "Custom policy" : "Create a custom policy"}
                  </span>
                </SelectItem>
              </SelectContent>
            </Select>
            {access.kind === "custom" ? (
              <Button
                type="button"
                variant="outline"
                size="md"
                disabled={disabled}
                onClick={openPanel}
              >
                Edit
              </Button>
            ) : null}
          </div>
        )}
      </FormField>
      <CustomPolicyPanel
        key={panel.key}
        open={panel.open}
        initial={access.kind === "custom" ? access.policies : []}
        workspaceId={workspaceId}
        grantable={grantable}
        onCancel={closePanel}
        onSave={(policies) => {
          onChange({ kind: "custom", policies });
          closePanel();
        }}
      />
    </>
  );
}

function ChosenLabel({ id, label }: { id: TemplateId; label: string }) {
  return (
    <span className="flex items-center gap-2">
      <span className="flex text-gray-11 [&>svg]:size-3.5">{TEMPLATE_ICONS[id]}</span>
      {label}
    </span>
  );
}

function CustomPolicyPanel({
  open,
  initial,
  workspaceId,
  grantable,
  onCancel,
  onSave,
}: {
  open: boolean;
  initial: Policy[];
  workspaceId: string;
  grantable: readonly string[];
  onCancel: () => void;
  onSave: (policies: Policy[]) => void;
}) {
  const [attempted, setAttempted] = useState(false);
  const editorCatalogue = useMemo(
    () => ({
      catalogues: grantableCatalogues(CATALOGUES, workspaceId, grantable),
      templates: TEMPLATES.filter((template) =>
        canGrant(grantable, workspaceId, template.materialise()),
      ),
    }),
    [workspaceId, grantable],
  );
  const firstScope =
    RESOURCE_SCOPES.find((scope) => catalogueRows(editorCatalogue.catalogues[scope]).length > 0) ??
    "workspace";
  const { form, bodyRef, submit } = useRootKeyPolicyForm(
    {
      name: "CLI",
      ipAllowlist: "",
      policies: initial.length > 0 ? initial : [newPolicy(firstScope)],
    },
    ({ policies }) => {
      if (!canGrant(grantable, workspaceId, policies)) {
        setAttempted(true);
        return;
      }
      onSave(policies);
    },
  );

  const draft = form.watch("policies") ?? [];
  const blocked = attempted ? uncoveredPermissions(grantable, buildUrns(workspaceId, draft)) : [];

  return (
    <SlidePanel isOpen={open} onClose={onCancel} widthClassName="w-192">
      <SlidePanelHeader className="items-center">
        <SlidePanelTitle>Custom policy</SlidePanelTitle>
        <SlidePanelCloseButton />
      </SlidePanelHeader>
      <SlidePanelContent>
        <FormProvider {...form}>
          <form
            onSubmit={(event) => {
              event.stopPropagation();
              void submit(event);
            }}
            className="flex h-full flex-col"
          >
            <div ref={bodyRef} className="flex-1 overflow-y-auto px-6 py-3">
              <EditorCatalogueProvider value={editorCatalogue}>
                <KeyFields nameHidden>
                  <PolicyList />
                </KeyFields>
              </EditorCatalogueProvider>
            </div>
            <SlidePanelFooter className="flex items-center justify-between gap-2">
              <p role="alert" className="min-w-0 text-sm text-error-11">
                {blocked.length > 0
                  ? `Your role can't grant: ${blocked.map(describeGrant).join(", ")}`
                  : null}
              </p>
              <div className="flex items-center gap-2">
                <Button type="button" variant="outline" size="md" onClick={onCancel}>
                  Cancel
                </Button>
                <Button type="submit" variant="primary" size="md">
                  Done
                </Button>
              </div>
            </SlidePanelFooter>
          </form>
        </FormProvider>
      </SlidePanelContent>
    </SlidePanel>
  );
}

function describeGrant(grant: string): string {
  const { path, action } = grantLabel(grant);
  return path === null ? action : `${action} on ${path}`;
}
