"use client";
import { WorkspaceNavbar } from "../workspace-navbar";
import { RootKeysListControlCloud } from "./components/control-cloud";
import { RootKeysListControls } from "./components/controls";
import { RootKeysList } from "./components/table/root-keys-list";

export default function RootKeysPage() {
  return (
    <div>
      <WorkspaceNavbar
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
