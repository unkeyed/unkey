import { CopyInput, SettingsRow } from "@unkey/ui";

export const CopyApiId = ({ apiId }: { apiId: string }) => {
  return (
    <SettingsRow title="API ID" description="An identifier for the API, used in some API calls.">
      <CopyInput value={apiId} aria-label="API ID" />
    </SettingsRow>
  );
};
