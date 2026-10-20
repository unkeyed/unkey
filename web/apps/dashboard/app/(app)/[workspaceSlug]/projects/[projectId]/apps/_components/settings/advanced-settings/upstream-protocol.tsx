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
import { Controller } from "react-hook-form";
import { z } from "zod";
import { useSettingField } from "../hooks/use-setting-form";

const PROTOCOLS: Record<EnvironmentSettings["upstreamProtocol"], { label: string; hint: string }> =
  {
    http1: { label: "HTTP/1.1", hint: "Works with every server." },
    h2c: { label: "HTTP/2 (h2c)", hint: "Cleartext HTTP/2. For gRPC and streaming." },
  };

export function UpstreamProtocol() {
  const { form, formProps } = useSettingField("upstreamProtocol", z.enum(["http1", "h2c"]));

  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Upstream protocol</SettingsRowTitle>
          <SettingsRowDescription>
            Protocol Unkey uses to reach your app.{" "}
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
              <RadioGroup
                aria-label="Upstream protocol"
                value={field.value}
                onValueChange={field.onChange}
                className="grid max-w-(--setting-w) gap-3 sm:grid-cols-2"
              >
                {Object.entries(PROTOCOLS).map(([value, protocol]) => (
                  <Radio.Root
                    key={value}
                    value={value}
                    className="group flex items-start gap-3 rounded-lg border border-grayA-5 p-3.5 text-left transition-colors hover:border-grayA-7 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7 data-checked:border-gray-12 data-checked:bg-grayA-2 data-checked:ring-1 data-checked:ring-gray-12"
                  >
                    <span className="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border border-grayA-7 group-data-checked:border-gray-12 group-data-checked:bg-gray-12">
                      <Radio.Indicator className="size-1.5 rounded-full bg-gray-1" />
                    </span>
                    <span className="flex flex-col gap-1">
                      <span className="font-mono text-sm font-medium text-gray-12">
                        {protocol.label}
                      </span>
                      <span className="text-xs leading-5 text-gray-11">{protocol.hint}</span>
                    </span>
                  </Radio.Root>
                ))}
              </RadioGroup>
            )}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}
