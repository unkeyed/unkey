"use client";

import { CircleInfo, Trash, XMark } from "@unkey/icons";
import {
  Button,
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { instanceLabels, useScopeInstances } from "../hooks/use-scope-instances";
import { CATALOGUES, catalogueFor } from "../lib/catalogue";
import { RESOURCE_SCOPES, type ResourceScope } from "../lib/catalogue.types";
import { type Policy, grantsPreview, newPolicy, policyError, policySummary } from "../lib/policy";
import { InstancePicker } from "./instance-picker";
import { PermissionCatalogue } from "./permission-catalogue";

const SCOPE_ITEMS: { value: ResourceScope; label: string }[] = RESOURCE_SCOPES.map((scope) => ({
  value: scope,
  label: CATALOGUES[scope].label,
}));

type PolicyCardProps = {
  policy: Policy;
  collapsed: boolean;
  showError: boolean;
  debug: boolean;
  onChange: (policy: Policy) => void;
  onRemove: () => void;
  onCollapsedChange: (collapsed: boolean) => void;
};

export function PolicyCard({
  policy,
  collapsed,
  showError,
  debug,
  onChange,
  onRemove,
  onCollapsedChange,
}: PolicyCardProps) {
  const catalogue = catalogueFor(policy.scope);
  const { instances, isLoading } = useScopeInstances(policy.scope);
  const error = policyError(policy);

  if (collapsed) {
    const summary = policySummary(policy, instanceLabels(instances));
    const { shown, more } = grantsPreview(summary.grants);

    return (
      <section className="flex items-center gap-2 rounded-lg border border-grayA-4 bg-white p-4 dark:bg-black">
        <button
          type="button"
          onClick={() => onCollapsedChange(false)}
          className="flex min-w-0 flex-1 cursor-pointer flex-col items-start gap-1 text-left"
        >
          <span className="w-full truncate text-[13px] text-accent-12">{summary.scopeLine}</span>
          {error ? (
            <span className="text-xs text-error-11">{error}</span>
          ) : (
            <span className="w-full truncate text-xs text-gray-10">
              {shown.join(", ")}
              {more > 0 ? ` +${more} more…` : ""}
            </span>
          )}
        </button>
        <RemovePolicyButton onRemove={onRemove} />
      </section>
    );
  }

  return (
    <section className="flex flex-col gap-4 rounded-lg border border-grayA-4 bg-white p-4 dark:bg-black">
      <div className="flex items-center gap-2">
        <span className="text-[13px] text-accent-12">Edit policy</span>
        <div className="ml-auto flex items-center gap-1">
          <RemovePolicyButton onRemove={onRemove} />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            aria-label="Close policy"
            className="size-8 shrink-0 justify-center rounded-lg px-0 text-gray-11 hover:bg-grayA-3 hover:text-gray-12"
            onClick={() => onCollapsedChange(true)}
          >
            <XMark />
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex h-5 items-center gap-2">
          <span className="text-[13px] text-gray-11">Scope</span>
          <InfoTooltip
            content="Choose which resources these privileges apply to."
            position={{ side: "right" }}
          >
            <CircleInfo iconSize="sm-regular" className="shrink-0 text-gray-9" />
          </InfoTooltip>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex-1">
            <Select
              value={policy.scope}
              items={SCOPE_ITEMS}
              onValueChange={(scope) => {
                if (scope) {
                  onChange(newPolicy(scope));
                }
              }}
            >
              <SelectTrigger aria-label="Resource type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SCOPE_ITEMS.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {catalogue.instanceNoun === null ? null : (
            <div className="flex-1">
              <InstancePicker
                noun={catalogue.instanceNoun}
                instances={instances}
                isLoading={isLoading}
                value={policy.instances}
                onChange={(next) => onChange({ ...policy, instances: next })}
              />
            </div>
          )}
        </div>
      </div>

      {showError && error ? <span className="text-xs text-error-11">{error}</span> : null}

      <PermissionCatalogue
        catalogue={catalogue}
        instances={policy.instances}
        debug={debug}
        value={policy.selection}
        onChange={(selection) => onChange({ ...policy, selection })}
      />
    </section>
  );
}

function RemovePolicyButton({ onRemove }: { onRemove: () => void }) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-label="Delete policy"
      className="size-8 shrink-0 justify-center rounded-lg px-0 text-gray-11 hover:bg-error-3 hover:text-error-11"
      onClick={onRemove}
    >
      <Trash />
    </Button>
  );
}
