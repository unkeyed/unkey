import { SettingsGroup, SettingsGroupContent, SettingsGroupTitle } from "@unkey/ui";
import { OpenapiSpecPath } from "./openapi-spec-path";
import { UpstreamProtocol } from "./upstream-protocol";

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
