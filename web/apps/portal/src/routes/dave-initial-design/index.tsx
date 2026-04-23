import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { KeysTable } from "~/components/keys-table/keys-table";
import { PortalFooter } from "~/components/portal-footer";
import { PortalLogoHeader } from "~/components/portal-logo-header";
import { type Key, seedBranding, seedKeys } from "./-seed";

type CSSVarStyle = React.CSSProperties & { [K in `--${string}`]?: string };

export const Route = createFileRoute("/dave-initial-design/")({
  component: Preview,
});

function Preview() {
  const [keys, setKeys] = useState<Key[]>(seedKeys);

  const rootStyle: CSSVarStyle = {
    "--portal-bg": seedBranding.backgroundColor,
    "--portal-btn": seedBranding.buttonColor,
  };

  return (
    <div style={rootStyle} className="flex min-h-screen flex-col bg-background">
      <PortalLogoHeader branding={seedBranding} />
      <main className="flex-1">
        <div className="mx-auto max-w-5xl px-8 py-12">
          <KeysTable
            keys={keys}
            onCreate={() => console.log("create")}
            onDelete={(id) => setKeys((prev) => prev.filter((k) => k.id !== id))}
            onEditExpiration={(id) => console.log("edit", id)}
            onRotate={(id) => console.log("rotate", id)}
          />
        </div>
      </main>
      <PortalFooter />
    </div>
  );
}
