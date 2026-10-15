import {
  Button,
  FormInput,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { useState } from "react";

export default function SlidePanelDirty() {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("Acme API");

  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        Rename app
      </Button>
      <SlidePanel
        isOpen={open}
        onClose={() => setOpen(false)}
        onExitComplete={() => setName("Acme API")}
        dirty={name !== "Acme API"}
        widthClassName="w-96"
        fitContent
      >
        <SlidePanelHeader>
          <SlidePanelTitle>Rename app</SlidePanelTitle>
          <SlidePanelCloseButton />
        </SlidePanelHeader>
        <SlidePanelContent className="px-6 pb-6">
          <FormInput label="Name" value={name} onChange={(e) => setName(e.target.value)} />
        </SlidePanelContent>
      </SlidePanel>
    </>
  );
}
