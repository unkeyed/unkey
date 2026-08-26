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
import { instanceLabels, useScopeInstances } from "./hooks/use-scope-instances";
import { InstancePicker } from "./instance-picker";
import { CATALOGUES } from "./lib/catalogue";
import { RESOURCE_SCOPES, type ResourceScope } from "./lib/catalogue.types";
import { type Policy, newPolicy, policyError } from "./lib/policy";
import { policySummary } from "./lib/policy-view";
import { PermissionCatalogue } from "./permission-catalogue";

const GRANT_PREVIEW_LIMIT = 3;

const SCOPE_ITEMS: { value: ResourceScope; label: string }[] = RESOURCE_SCOPES.map((scope) => ({
  value: scope,
  label: CATALOGUES[scope].label,
}));

type PolicySummaryRowProps = {
  policy: Policy;
  onExpand: () => void;
  onRemove: () => void;
};

export function PolicySummaryRow({ policy, onExpand, onRemove }: PolicySummaryRowProps) {
  const { instances } = useScopeInstances(policy.scope);
  const error = policyError(policy);
  const summary = policySummary(policy, instanceLabels(instances));
  const shown = summary.grants.slice(0, GRANT_PREVIEW_LIMIT);
  const more = Math.max(summary.grants.length - GRANT_PREVIEW_LIMIT, 0);

  return (
    <section className="flex items-center gap-2 rounded-lg border border-grayA-4 dark:border-grayA-5 bg-white p-4 dark:bg-black">
      <button
        type="button"
        onClick={onExpand}
        className="flex min-w-0 flex-1 cursor-pointer flex-col items-start gap-1 text-left"
      >
        <span className="w-full truncate text-[13px] font-medium text-accent-12">
          {summary.scopeLine}
        </span>
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

type PolicyEditorProps = {
  policy: Policy;
  error?: string;
  onChange: (policy: Policy) => void;
  onCollapse: () => void;
  onRemove: () => void;
};

export function PolicyEditor({ policy, error, onChange, onCollapse, onRemove }: PolicyEditorProps) {
  const catalogue = CATALOGUES[policy.scope];
  const { instances, isLoading } = useScopeInstances(policy.scope);

  return (
    <section className="flex flex-col gap-4 rounded-lg border border-grayA-4 dark:border-grayA-5 bg-white p-4 dark:bg-black">
      <div className="flex items-center gap-2">
        <span className="text-[13px] font-medium text-accent-12">Edit policy</span>
        <div className="ml-auto flex items-center gap-1">
          <RemovePolicyButton onRemove={onRemove} />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            aria-label="Close policy"
            className="size-8 shrink-0 justify-center rounded-lg px-0 text-gray-11 hover:bg-grayA-3 hover:text-gray-12"
            onClick={onCollapse}
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

      {error ? <span className="text-xs text-error-11">{error}</span> : null}

      <PermissionCatalogue
        key={policy.scope}
        catalogue={catalogue}
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
