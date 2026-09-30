import { FormInput, SettingsGroup, SettingsRow } from "@unkey/ui";

export default function SettingsRowExample() {
  return (
    <SettingsGroup title="Build settings">
      <SettingsRow
        title="Build command"
        description="Override the auto-detected build command. Useful for monorepos."
      >
        <FormInput placeholder="pnpm build" defaultValue="pnpm build --filter api" />
      </SettingsRow>
      <SettingsRow title="Root directory" description="Directory the build runs in.">
        <FormInput placeholder="./" defaultValue="./services/api" />
      </SettingsRow>
    </SettingsGroup>
  );
}
