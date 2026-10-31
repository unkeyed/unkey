"use client";

import { IconSlidersOutline18 } from "@unkey/icons";
import { CollapsibleItem } from "@unkey/ui";
import { useState } from "react";
import { type UseFormReturn, get } from "react-hook-form";
import { useEnvironmentSettings } from "../environment-provider";
import { HealthcheckTiming } from "./healthcheck-fields";
import { ADVANCED_PATHS, type RuntimeValues, hasAdvancedOverrides, readRuntime } from "./model";
import { ShutdownSignalField } from "./shutdown-signal-field";

export function AdvancedFields({ form }: { form: UseFormReturn<RuntimeValues> }) {
  const { settings } = useEnvironmentSettings();
  const [open, setOpen] = useState(() => hasAdvancedOverrides(readRuntime(settings)));
  const hiddenError = ADVANCED_PATHS.some((path) => get(form.formState.errors, path));

  if (hiddenError && !open) {
    setOpen(true);
  }

  return (
    <CollapsibleItem
      icon={<IconSlidersOutline18 />}
      title="Advanced configuration"
      description="Shutdown signal and health check timing"
      open={open}
      onOpenChange={setOpen}
    >
      <div className="flex flex-col gap-5 px-4 pt-4 pb-5">
        <ShutdownSignalField form={form} />
        <HealthcheckTiming form={form} />
      </div>
    </CollapsibleItem>
  );
}
