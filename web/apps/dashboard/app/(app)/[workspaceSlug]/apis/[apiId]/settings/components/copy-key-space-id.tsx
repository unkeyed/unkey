import { CopyInput, SettingsRow } from "@unkey/ui";

export const CopyKeySpaceId = ({ keySpaceId }: { keySpaceId: string }) => {
  return (
    <SettingsRow title="KeySpace ID" description="Identifier for the underlying keyspace.">
      <CopyInput value={keySpaceId} aria-label="KeySpace ID" />
    </SettingsRow>
  );
};
