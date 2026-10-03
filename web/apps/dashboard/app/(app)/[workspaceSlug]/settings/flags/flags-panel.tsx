"use client";

import type { WorkspaceFlag } from "@/lib/workspace-flags-api";
import { Badge, Button, Checkbox, FormInput, SettingCard, SettingCardGroup } from "@unkey/ui";
import { useState } from "react";

type Props = {
  flags: WorkspaceFlag[];
  isAdmin: boolean;
  onSet: (slug: string, value: boolean | string | number) => Promise<void>;
  onRemove: (slug: string) => Promise<void>;
};

export function FlagsPanel({ flags, isAdmin, onSet, onRemove }: Props) {
  if (flags.length === 0) {
    return (
      <div className="rounded-lg border border-grayA-4 p-8 text-center">
        <h2 className="text-sm font-medium text-gray-12">No flags available</h2>
        <p className="mt-2 text-sm text-gray-11">
          Workspace flags will appear here when Unkey makes them available.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      {!isAdmin && (
        <p className="text-sm text-gray-11">Only workspace admins can change flag overrides.</p>
      )}
      <SettingCardGroup>
        {flags.map((flag) => (
          <FlagCard
            key={`${flag.slug}:${flag.hasOverride}:${JSON.stringify(flag.value)}`}
            flag={flag}
            isAdmin={isAdmin}
            onSet={onSet}
            onRemove={onRemove}
          />
        ))}
      </SettingCardGroup>
    </div>
  );
}

function FlagCard({
  flag,
  isAdmin,
  onSet,
  onRemove,
}: Omit<Props, "flags"> & { flag: WorkspaceFlag }) {
  const [input, setInput] = useState(String(flag.value));
  const [pending, setPending] = useState<"save" | "remove" | null>(null);
  const [error, setError] = useState<string>();
  const disabled = !isAdmin || !flag.allowOptIn || pending !== null;
  const inputId = `flag-${flag.slug}`;

  async function save() {
    let value: boolean | string | number = input;
    if (flag.type === "boolean") {
      value = input === "true";
    } else if (flag.type === "number") {
      value = Number(input);
      if (input.trim() === "" || !Number.isFinite(value)) {
        setError("Enter a finite number.");
        return;
      }
    }
    await mutate("save", () => onSet(flag.slug, value));
  }

  async function mutate(action: "save" | "remove", operation: () => Promise<void>) {
    setPending(action);
    setError(undefined);
    try {
      await operation();
    } catch (error) {
      setError(error instanceof Error ? error.message : "We couldn't update this flag. Try again.");
    } finally {
      setPending(null);
    }
  }

  return (
    <SettingCard
      title={
        <span className="flex flex-wrap items-center gap-2">
          <span className="font-mono">{flag.slug}</span>
          <Badge variant={flag.hasOverride ? "success" : "secondary"}>
            {flag.hasOverride ? "Override" : "Default"}
          </Badge>
        </span>
      }
      description={
        <span className="flex flex-col gap-2">
          <span>{flag.description}</span>
          <span className="text-xs">
            {flag.type} · Default: <code>{JSON.stringify(flag.defaultValue)}</code>
          </span>
        </span>
      }
      contentWidth="w-full lg:w-80"
    >
      <form
        className="flex w-full flex-col gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        {flag.type === "boolean" ? (
          <div className="flex items-center gap-2">
            <Checkbox
              id={inputId}
              aria-label={`Enable ${flag.slug}`}
              checked={input === "true"}
              disabled={disabled}
              onCheckedChange={(checked) => {
                setInput(String(checked));
              }}
            />
            <label htmlFor={inputId} className="text-sm text-gray-12">
              Enabled
            </label>
          </div>
        ) : (
          <FormInput
            id={inputId}
            label="Workspace value"
            type={flag.type === "number" ? "number" : "text"}
            step="any"
            maxLength={4096}
            value={input}
            disabled={disabled}
            onChange={(event) => {
              setInput(event.target.value);
            }}
          />
        )}
        <div className="flex flex-wrap gap-2">
          <Button
            type="submit"
            variant="primary"
            disabled={disabled || (flag.hasOverride && input === String(flag.value))}
            loading={pending === "save"}
          >
            {flag.hasOverride ? "Save changes" : "Set override"}
          </Button>
          {flag.hasOverride && (
            <Button
              type="button"
              variant="outline"
              disabled={!isAdmin || !flag.allowOptOut || pending !== null}
              loading={pending === "remove"}
              onClick={() => void mutate("remove", () => onRemove(flag.slug))}
            >
              Use default
            </Button>
          )}
        </div>
        {!flag.allowOptIn && (
          <p className="text-xs text-gray-11">Value changes are managed by Unkey.</p>
        )}
        {!flag.allowOptOut && (flag.hasOverride || flag.allowOptIn) && (
          <p className="text-xs text-gray-11">
            {flag.hasOverride
              ? "Only Unkey can remove this override."
              : "After saving, only Unkey can remove this override."}
          </p>
        )}
        {error && (
          <p role="alert" className="text-sm text-error-11">
            {error}
          </p>
        )}
      </form>
    </SettingCard>
  );
}
