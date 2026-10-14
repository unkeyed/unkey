import {
  FormInput,
  SettingsForm,
  SettingsGroupCollapsible,
  SettingsGroups,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { useState } from "react";

export default function CollapsibleSettingsGroup() {
  const [saved, setSaved] = useState("8080");
  const [port, setPort] = useState(saved);
  const dirty = port !== saved;

  return (
    <SettingsGroups pendingNote="Applies on the next deploy" className="p-0">
      <SettingsGroupCollapsible
        title="Runtime"
        description="The defaults suit most apps."
        summary={`Port ${saved}`}
      >
        <SettingsForm
          dirty={dirty}
          saveState={dirty ? { status: "ready" } : { status: "disabled" }}
          onSubmit={(e) => {
            e.preventDefault();
            setSaved(port);
          }}
        >
          <SettingsRow>
            <SettingsRowHeader>
              <SettingsRowTitle>Port</SettingsRowTitle>
              <SettingsRowDescription>The port your app listens on.</SettingsRowDescription>
            </SettingsRowHeader>
            <SettingsRowContent>
              <FormInput
                aria-label="Port"
                className="max-w-(--setting-w)"
                value={port}
                onChange={(e) => setPort(e.target.value)}
              />
            </SettingsRowContent>
          </SettingsRow>
        </SettingsForm>
      </SettingsGroupCollapsible>
    </SettingsGroups>
  );
}
