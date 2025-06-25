"use client";
import { RootKeysList } from "./components/table/root-keys-list";
import { Navigation } from "./navigation";

export default function RootKeysPage() {
  return (
    <div>
      <Navigation
        workspace={{
          id: "will-add-soon",
          name: "will-add-soon",
        }}
        activePage={{
          href: "root-keys",
          text: "Root Keys",
        }}
      />
      <div className="flex flex-col">
        <RootKeysList />
      </div>
    </div>
  );
}
