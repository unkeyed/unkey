import {
  type SaveState,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import type React from "react";

type FormSettingCardProps = {
  title: string;
  description?: React.ReactNode;
  dirty: boolean;
  saveState: SaveState;
  onSubmit: React.FormEventHandler<HTMLFormElement>;
  autoSave?: boolean;
  children: React.ReactNode;
};

export function FormSettingCard({
  title,
  description,
  children,
  ...formProps
}: FormSettingCardProps) {
  return (
    <SettingsForm {...formProps}>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>{title}</SettingsRowTitle>
          {description ? <SettingsRowDescription>{description}</SettingsRowDescription> : null}
        </SettingsRowHeader>
        <SettingsRowContent>
          <div className="flex flex-col gap-2">{children}</div>
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}
