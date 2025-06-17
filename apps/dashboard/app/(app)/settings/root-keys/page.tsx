"use client";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { RootKeysList } from "./components/table/root-keys-list";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  return (
    <div>
      <Navigation
        workspace={{
          id: "",
          name: "",
        }}
        activePage={{
          href: "root-keys",
          text: "Root Keys",
        }}
      />
      <div className="flex flex-col">
        <RootKeysListControls />
        <RootKeysListControlCloud />
        <RootKeysList />
      </div>
    </div>
  );
}
