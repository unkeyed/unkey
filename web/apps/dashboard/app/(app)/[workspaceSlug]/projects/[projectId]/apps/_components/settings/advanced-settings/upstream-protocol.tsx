"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";
import {
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { useId } from "react";
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
  const descriptionId = useId();

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
            render={({ field }) => (
              <div className="flex max-w-xs flex-col gap-2">
                <RadioGroup
                  aria-label="Upstream protocol"
                  value={field.value}
                  onValueChange={field.onChange}
                  className="grid grid-cols-2 gap-1 rounded-lg bg-grayA-3 p-1"
                >
                  {Object.entries(PROTOCOLS).map(([value, protocol]) => (
                    <Radio.Root
                      key={value}
                      value={value}
                      aria-describedby={`${descriptionId}-${value}`}
                      className="flex min-h-8 cursor-pointer items-center justify-center rounded-md border border-transparent px-3 font-mono text-xs text-gray-11 transition-colors hover:text-gray-12 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-8 focus-visible:ring-offset-2 focus-visible:ring-offset-background data-checked:border-grayA-5 data-checked:bg-raised data-checked:text-gray-12 data-checked:shadow-xs"
                    >
                      {protocol.label}
                    </Radio.Root>
                  ))}
                </RadioGroup>
                {Object.entries(PROTOCOLS).map(([value, protocol]) => (
                  <p
                    key={value}
                    id={`${descriptionId}-${value}`}
                    hidden={value !== field.value}
                    className="text-xs leading-4 text-gray-11"
                  >
                    {protocol.hint}
                  </p>
                ))}
              </div>
            )}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}
