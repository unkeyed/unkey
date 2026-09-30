"use client";

import {
  IconCheckOutline18,
  IconCubeOutline18,
  IconMagnifierOutline18,
  IconPlusOutline18,
} from "@unkey/icons";
import {
  Button,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  Popover,
  PopoverContent,
  PopoverTrigger,
  cn,
} from "@unkey/ui";
import { useLayoutEffect, useRef, useState } from "react";
import type { BindingTargets, Environment, TargetRule } from "./binding-rules";

export function AddBindingPicker({
  open,
  onOpenChange,
  apps,
  environments,
  requiresEnvironment,
  pending,
  onAdd,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  apps: BindingTargets["apps"];
  environments: Environment[];
  requiresEnvironment: boolean;
  pending: boolean;
  onAdd: (appId: string, rule: TargetRule) => void;
}) {
  const [step, setStep] = useState<"resource" | "app">("resource");
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [environmentId, setEnvironmentId] = useState("");
  const searchInput = useRef<HTMLInputElement>(null);
  const selected = apps.find((app) => app.id === selectedId);
  const query = search.trim().toLowerCase();
  const visible = apps.filter((app) => `${app.name} ${app.slug}`.toLowerCase().includes(query));
  const targetEnvironments = environments.filter((env) => env.appId === selected?.id);
  const canAdd =
    selected !== undefined &&
    (!requiresEnvironment || targetEnvironments.some((env) => env.id === environmentId));

  useLayoutEffect(() => {
    if (open && step === "app") {
      searchInput.current?.focus();
    }
  }, [open, step]);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (next) {
          setStep("resource");
          setSearch("");
          setSelectedId(null);
          setEnvironmentId("");
        }
      }}
    >
      <PopoverTrigger
        disabled={pending}
        className="flex h-9 w-full items-center gap-2 rounded-lg border border-dashed border-gray-6 px-3 text-sm text-gray-11 hover:border-gray-8 hover:bg-gray-3 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-8 disabled:opacity-50"
      >
        <IconPlusOutline18 className="size-4" />
        Add binding
      </PopoverTrigger>
      <PopoverContent
        side="bottom"
        align="start"
        sideOffset={12}
        className="w-72 max-w-[calc(100vw-2rem)] p-2"
        aria-label={step === "resource" ? "Add binding" : "Choose an app"}
      >
        {step === "resource" ? (
          <button
            type="button"
            className="flex w-full items-center gap-3 rounded-md p-2 text-left hover:bg-gray-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-gray-8"
            onClick={() => setStep("app")}
          >
            <IconCubeOutline18 className="size-4 text-gray-11" />
            <span>
              <span className="block text-sm font-medium text-gray-12">App</span>
              <span className="block text-xs text-gray-9">Private connection</span>
            </span>
          </button>
        ) : (
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (selected && canAdd && !pending) {
                onAdd(
                  selected.id,
                  requiresEnvironment
                    ? {
                        targetType: "environment",
                        targetEnvironmentId: environmentId,
                      }
                    : { targetType: "automatic" },
                );
              }
            }}
          >
            <h2 className="px-1 pb-3 pt-1 text-sm font-medium">Choose an app</h2>
            <InputGroup className="h-8">
              <InputGroupAddon>
                <IconMagnifierOutline18 className="size-4 text-gray-9" />
              </InputGroupAddon>
              <InputGroupInput
                ref={searchInput}
                aria-label="Search apps"
                placeholder="Search apps…"
                value={search}
                disabled={pending}
                onChange={(event) => {
                  setSearch(event.target.value);
                  setSelectedId(null);
                  setEnvironmentId("");
                }}
              />
            </InputGroup>
            <div
              className="my-2 max-h-48 overflow-y-auto"
              role="radiogroup"
              aria-label="Available apps"
            >
              {visible.map((app) => (
                <label
                  key={app.id}
                  className={cn(
                    "flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm hover:bg-gray-3 has-focus-visible:outline has-focus-visible:outline-2 has-focus-visible:outline-gray-8",
                    selectedId === app.id && "bg-gray-3",
                  )}
                >
                  <input
                    type="radio"
                    name="binding-app"
                    value={app.id}
                    checked={selectedId === app.id}
                    disabled={pending}
                    className="sr-only"
                    onChange={() => {
                      setSelectedId(app.id);
                      setEnvironmentId("");
                    }}
                  />
                  <IconCubeOutline18 className="size-4 shrink-0 text-gray-9" />
                  <span className="min-w-0 flex-1 truncate">{app.name}</span>
                  {selectedId === app.id && <IconCheckOutline18 className="size-4 shrink-0" />}
                </label>
              ))}
              {visible.length === 0 && (
                <p className="px-2 py-3 text-xs text-gray-9">
                  {apps.length === 0 ? "All apps are already connected." : "No matching apps."}
                </p>
              )}
            </div>
            {selected && requiresEnvironment && (
              <label className="mb-3 flex flex-col gap-1.5 px-1 text-xs text-gray-11">
                Target environment
                <select
                  className="h-8 rounded-md border border-gray-5 bg-background px-2"
                  value={environmentId}
                  disabled={pending}
                  onChange={(event) => setEnvironmentId(event.target.value)}
                >
                  <option value="">Choose an environment</option>
                  {targetEnvironments.map((env) => (
                    <option key={env.id} value={env.id}>
                      {env.slug}
                    </option>
                  ))}
                </select>
              </label>
            )}
            <Button
              type="submit"
              className="w-full"
              disabled={!canAdd || pending}
              loading={pending}
            >
              Add binding
            </Button>
          </form>
        )}
      </PopoverContent>
    </Popover>
  );
}
