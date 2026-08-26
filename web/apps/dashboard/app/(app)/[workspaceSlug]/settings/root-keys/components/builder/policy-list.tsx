"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useReducer } from "react";
import { Controller, useFormContext, useFormState } from "react-hook-form";
import { type Policy, isPolicyComplete } from "./lib/policy";
import { PolicyEditor, PolicySummaryRow } from "./policy-card";
import type { RootKeyFormValues } from "./schema";
import { TemplateGallery } from "./template-gallery";

type PolicyEntry = {
  id: string;
  policy: Policy;
  collapsed: boolean;
};

type PolicyListState = {
  entries: PolicyEntry[];
  gallery: boolean;
};

type PolicyListAction =
  | { type: "add"; entries: PolicyEntry[] }
  | { type: "update"; id: string; policy: Policy }
  | { type: "remove"; id: string }
  | { type: "collapse"; id: string; collapsed: boolean }
  | { type: "openGallery" }
  | { type: "closeGallery" };

// Ids are minted here rather than in the reducer so the reducer stays pure and
// a policy arrives with the collapse state it keeps: templates land complete and
// therefore folded, "Start new" lands empty and therefore open.
function entryOf(policy: Policy): PolicyEntry {
  return { id: crypto.randomUUID(), policy, collapsed: isPolicyComplete(policy) };
}

function reduce(state: PolicyListState, action: PolicyListAction): PolicyListState {
  switch (action.type) {
    case "add":
      return { entries: [...state.entries, ...action.entries], gallery: false };
    case "update":
      return {
        ...state,
        entries: state.entries.map((entry) =>
          entry.id === action.id ? { ...entry, policy: action.policy } : entry,
        ),
      };
    case "remove":
      return { ...state, entries: state.entries.filter((entry) => entry.id !== action.id) };
    case "collapse":
      return {
        ...state,
        entries: state.entries.map((entry) =>
          entry.id === action.id ? { ...entry, collapsed: action.collapsed } : entry,
        ),
      };
    case "openGallery":
      return {
        entries: state.entries.map((entry) => ({ ...entry, collapsed: true })),
        gallery: true,
      };
    case "closeGallery":
      return { ...state, gallery: false };
  }
}

export function PolicyList() {
  const { control } = useFormContext<RootKeyFormValues>();

  return (
    <Controller
      control={control}
      name="policies"
      render={({ field }) => <PolicyListBody policies={field.value} onChange={field.onChange} />}
    />
  );
}

type PolicyListBodyProps = {
  policies: Policy[];
  onChange: (policies: Policy[]) => void;
};

function PolicyListBody({ policies, onChange }: PolicyListBodyProps) {
  const { control } = useFormContext<RootKeyFormValues>();
  const { errors } = useFormState({ control, name: "policies" });
  const [state, dispatch] = useReducer(reduce, policies, (initial) => ({
    entries: initial.map(entryOf),
    gallery: false,
  }));

  const send = (action: PolicyListAction) => {
    dispatch(action);
    if (action.type === "add" || action.type === "update" || action.type === "remove") {
      onChange(reduce(state, action).entries.map((entry) => entry.policy));
    }
  };

  const showGallery = state.gallery || state.entries.length === 0;

  return (
    <div className="flex flex-col gap-3">
      {state.entries.map((entry, index) =>
        entry.collapsed ? (
          <PolicySummaryRow
            key={entry.id}
            policy={entry.policy}
            onExpand={() => send({ type: "collapse", id: entry.id, collapsed: false })}
            onRemove={() => send({ type: "remove", id: entry.id })}
          />
        ) : (
          <PolicyEditor
            key={entry.id}
            policy={entry.policy}
            error={errors.policies?.[index]?.message}
            onChange={(policy) => send({ type: "update", id: entry.id, policy })}
            onCollapse={() => send({ type: "collapse", id: entry.id, collapsed: true })}
            onRemove={() => send({ type: "remove", id: entry.id })}
          />
        ),
      )}

      {errors.policies?.message ? (
        <span className="text-[13px] leading-5 text-error-11">{errors.policies.message}</span>
      ) : null}

      {showGallery ? (
        <TemplateGallery
          onPick={(picked) => send({ type: "add", entries: picked.map(entryOf) })}
          onCancel={state.entries.length > 0 ? () => send({ type: "closeGallery" }) : undefined}
        />
      ) : (
        <div>
          <Button
            type="button"
            variant="ghost"
            size="md"
            className="font-medium"
            onClick={() => send({ type: "openGallery" })}
          >
            <Plus />
            Add policy
          </Button>
        </div>
      )}
    </div>
  );
}
