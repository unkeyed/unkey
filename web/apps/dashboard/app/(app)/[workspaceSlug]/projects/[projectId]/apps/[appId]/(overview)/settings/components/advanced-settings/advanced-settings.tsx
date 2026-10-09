"use client";

import { SettingsGroup, SettingsGroupContent, SettingsGroupTitle } from "@unkey/ui";
import { OpenapiSpecPath } from "../../../../../_components/settings/advanced-settings/openapi-spec-path";
import { UpstreamProtocol } from "../../../../../_components/settings/advanced-settings/upstream-protocol";

export function AdvancedSettings() {
  return (
    <>
      <SettingsGroup>
        <SettingsGroupTitle>API</SettingsGroupTitle>
        <SettingsGroupContent>
          <OpenapiSpecPath />
        </SettingsGroupContent>
      </SettingsGroup>
      <SettingsGroup>
        <SettingsGroupTitle>Networking</SettingsGroupTitle>
        <SettingsGroupContent>
          <UpstreamProtocol />
        </SettingsGroupContent>
      </SettingsGroup>
    </>
  );
}
