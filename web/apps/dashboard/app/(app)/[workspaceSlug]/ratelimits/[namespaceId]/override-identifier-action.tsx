"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import dynamic from "next/dynamic";
import { IconPlusOutline18 } from "nucleo-ui-outline-18";
import { useState } from "react";

const IdentifierDialog = dynamic(
  () => import("./_components/identifier-dialog").then((mod) => mod.IdentifierDialog),
  {
    loading: () => null,
    ssr: false,
  },
);

export function OverrideIdentifierAction({ namespaceId }: { namespaceId: string }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <NavbarActionButton title="Override Identifier" onClick={() => setOpen(true)}>
        <IconPlusOutline18 />
        Override Identifier
      </NavbarActionButton>
      {open && (
        <IdentifierDialog onOpenChange={setOpen} isModalOpen={open} namespaceId={namespaceId} />
      )}
    </>
  );
}
