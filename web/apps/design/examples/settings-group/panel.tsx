import {
  Button,
  FormInput,
  SettingsForm,
  SettingsPanel,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { useState } from "react";

function PortForm({ saved, onSave }: { saved: string; onSave: (port: string) => void }) {
  const [port, setPort] = useState(saved);
  const dirty = port !== saved;

  return (
    <SettingsForm
      dirty={dirty}
      saveState={dirty ? { status: "ready" } : { status: "disabled" }}
      onSubmit={(e) => {
        e.preventDefault();
        onSave(port);
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
  );
}

export default function SettingsGroupInPanel() {
  const [open, setOpen] = useState(false);
  const [saved, setSaved] = useState("8080");
  const close = () => setOpen(false);

  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        Edit runtime (port {saved})
      </Button>
      <SettingsPanel isOpen={open} onClose={close} onSaved={close} title="Runtime">
        <PortForm saved={saved} onSave={setSaved} />
      </SettingsPanel>
    </>
  );
}
