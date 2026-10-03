"use client";

import { Switch } from "@/components/ui/switch";
import type { WorkspaceFlag } from "@/lib/workspace-flags-api";
import { SettingCard, SettingCardGroup } from "@unkey/ui";
import { useState } from "react";

type Props = {
  flags: WorkspaceFlag[];
  isAdmin: boolean;
  onSet: (slug: string, value: boolean) => Promise<void>;
};

export function FlagsPanel({ flags, isAdmin, onSet }: Props) {
  if (flags.length === 0) {
    return (
      <div className="rounded-lg border border-grayA-4 p-8 text-center">
        <h2 className="text-sm font-medium text-gray-12">No platform features available</h2>
        <p className="mt-2 text-sm text-gray-11">
          Platform features will appear here when Unkey makes them available.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      {!isAdmin && (
        <p className="text-sm text-gray-11">Only workspace admins can change platform features.</p>
      )}
      <SettingCardGroup>
        {flags.map((flag) => (
          <FlagCard key={flag.slug} flag={flag} isAdmin={isAdmin} onSet={onSet} />
        ))}
      </SettingCardGroup>
    </div>
  );
}

function FlagCard({ flag, isAdmin, onSet }: Omit<Props, "flags"> & { flag: WorkspaceFlag }) {
  const [pendingValue, setPendingValue] = useState<boolean | null>(null);
  const [error, setError] = useState<string>();
  const pending = pendingValue !== null;

  async function toggle(enabled: boolean) {
    setPendingValue(enabled);
    setError(undefined);
    try {
      await onSet(flag.slug, enabled);
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "We couldn't update this platform feature. Try again.",
      );
    } finally {
      setPendingValue(null);
    }
  }

  return (
    <SettingCard
      title={<span className="font-mono">{flag.slug}</span>}
      description={flag.description}
      contentWidth="w-full lg:w-64"
    >
      <div className="flex w-full flex-col items-end gap-3">
        <Switch
          aria-label={`Enable ${flag.slug}`}
          aria-busy={pending}
          checked={pendingValue ?? flag.value}
          disabled={!isAdmin || !flag.allowOptIn || pending}
          onCheckedChange={(enabled) => void toggle(enabled)}
        />
        {!flag.allowOptIn && (
          <p className="text-xs text-gray-11">This platform feature is managed by Unkey.</p>
        )}
        {error && (
          <p role="alert" className="text-sm text-error-11">
            {error}
          </p>
        )}
      </div>
    </SettingCard>
  );
}
