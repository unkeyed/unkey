import {
  FormInput,
  SettingsForm,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  formSaveState,
} from "@unkey/ui";
import { useState } from "react";

function TextSetting({
  title,
  description,
  initial,
  placeholder,
}: {
  title: string;
  description: string;
  initial: string;
  placeholder: string;
}) {
  const [saved, setSaved] = useState(initial);
  const [value, setValue] = useState(initial);
  const isDirty = value !== saved;

  return (
    <SettingsForm
      dirty={isDirty}
      saveState={formSaveState({ isSubmitting: false, isValid: value !== "", isDirty })}
      onSubmit={(e) => {
        e.preventDefault();
        setSaved(value);
      }}
    >
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>{title}</SettingsRowTitle>
          <SettingsRowDescription>{description}</SettingsRowDescription>
        </SettingsRowHeader>
        <SettingsRowContent>
          <FormInput
            aria-label={title}
            placeholder={placeholder}
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}

export default function SettingsRowExample() {
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Build settings</SettingsGroupTitle>
      <SettingsGroupContent>
        <TextSetting
          title="Build command"
          description="Override the auto-detected build command. Useful for monorepos."
          initial="pnpm build --filter api"
          placeholder="pnpm build"
        />
        <TextSetting
          title="Root directory"
          description="Directory the build runs in."
          initial="./services/api"
          placeholder="./"
        />
      </SettingsGroupContent>
    </SettingsGroup>
  );
}
