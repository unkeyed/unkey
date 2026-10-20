"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import {
  FormField,
  SegmentedControl,
  SegmentedControlItem,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { Controller } from "react-hook-form";
import { z } from "zod";
import { useSettingField } from "../hooks/use-setting-form";

export const PROTOCOLS: Record<
  EnvironmentSettings["upstreamProtocol"],
  { label: string; hint: string }
> = {
  http1: {
    label: "HTTP/1.1",
    hint: "The default for most applications. No special server configuration needed.",
  },
  h2c: {
    label: "HTTP/2 (h2c)",
    hint: "For gRPC and HTTP/2 servers. Your app must accept cleartext HTTP/2 connections.",
  },
};

export function UpstreamProtocol() {
  const { form, formProps } = useSettingField("upstreamProtocol", z.enum(["http1", "h2c"]));

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Upstream protocol</SettingsRowTitle>
          <SettingsRowDescription>
            How Unkey connects to your app.{" "}
            <a
              href="https://www.unkey.com/docs/platform/apps/settings#upstream-protocol"
              target="_blank"
              rel="noopener noreferrer"
              className="underline text-gray-11"
            >
              Learn more
            </a>
            .
          </SettingsRowDescription>
        </SettingsRowHeader>
        <SettingsRowContent>
          <Controller
            control={form.control}
            name="upstreamProtocol"
            render={({ field, fieldState }) => (
              <FormField
                className="max-w-xs"
                description={PROTOCOLS[field.value].hint}
                error={fieldState.error?.message}
              >
                {(control) => (
                  <SegmentedControl
                    id={control.id}
                    aria-label="Upstream protocol"
                    aria-describedby={control.describedBy}
                    aria-invalid={control.invalid}
                    name={field.name}
                    value={field.value}
                    onValueChange={field.onChange}
                    onBlur={field.onBlur}
                    inputRef={field.ref}
                  >
                    {Object.entries(PROTOCOLS).map(([value, protocol]) => (
                      <SegmentedControlItem key={value} value={value} className="font-mono text-xs">
                        {protocol.label}
                      </SegmentedControlItem>
                    ))}
                  </SegmentedControl>
                )}
              </FormField>
            )}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}
