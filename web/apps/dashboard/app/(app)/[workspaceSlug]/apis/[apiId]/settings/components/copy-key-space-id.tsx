import {
  CopyInput,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";

export const CopyKeySpaceId = ({ keySpaceId }: { keySpaceId: string }) => {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>KeySpace ID</SettingsRowTitle>
        <SettingsRowDescription>Identifier for the underlying keyspace.</SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        <CopyInput value={keySpaceId} aria-label="KeySpace ID" className="max-w-(--setting-w)" />
      </SettingsRowContent>
    </SettingsRow>
  );
};
