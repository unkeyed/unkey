"use client";

import type { WwwMode } from "@/lib/collections/deploy/edge-redirects.schema";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ArrowRight, ChevronDown } from "@unkey/icons";
import { Popover, PopoverContent, PopoverTrigger, toast } from "@unkey/ui";
import { useState } from "react";
import { useProjectData } from "../../../../data-provider";

type RedirectPickerProps = {
  /** This row's FQDN. The picker reads/writes the apex/www pair derived from it. */
  domain: string;
};

/**
 * Inline picker that shows what role this row plays in its apex/www pair
 * (primary, redirecting, or standalone) and lets the user change it.
 *
 * The trigger sits in the row's right-side cluster next to the status
 * badge — same visual weight as the existing badges and buttons. The
 * popover holds three concrete choices spelled out with the actual
 * hostnames so the user is never translating from "stripWww" to "OK so
 * www redirects to apex". Backend writes both rows in a transaction.
 */
export function RedirectPicker({ domain }: RedirectPickerProps) {
  const { projectId } = useProjectData();
  const utils = trpc.useUtils();
  const [open, setOpen] = useState(false);

  const { data } = trpc.deploy.edgeRedirects.get.useQuery({ projectId, domain }, { staleTime: 0 });

  const updateMutation = trpc.deploy.edgeRedirects.update.useMutation({
    onSuccess: async () => {
      // Invalidate every picker in this project — the mutation writes both
      // sides of the apex/www pair, so the sister row's picker needs to
      // refetch its joint state too. Filtering by projectId keeps us from
      // refetching unrelated projects' rows on the same screen.
      await utils.deploy.edgeRedirects.get.invalidate({ projectId });
    },
    onError: (err) => {
      toast.error("Failed to update", { description: err.message });
    },
  });

  if (!data) {
    return null;
  }

  const isApexRow = domain === data.apexDomain;
  const sister = isApexRow ? data.wwwDomain : data.apexDomain;
  const role = roleForRow({ mode: data.wwwMode, isApexRow, sister });

  const choose = (mode: WwwMode) => {
    setOpen(false);
    if (mode === data.wwwMode) {
      return;
    }
    updateMutation.mutate({ projectId, domain, wwwMode: mode });
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex items-center gap-1.5 h-7 px-2 rounded-md border text-[12px] font-medium",
            "border-grayA-4 hover:border-grayA-6 hover:bg-grayA-2 transition-colors",
            "text-gray-11 data-[state=open]:bg-grayA-3",
          )}
        >
          <RoleLabel role={role} />
          <ChevronDown className="size-3! text-gray-9" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[320px] p-1">
        <Choice
          selected={data.wwwMode === "stripWww"}
          disabled={!data.wwwExists}
          primary={data.apexDomain}
          redirecting={data.wwwDomain}
          onSelect={() => choose("stripWww")}
        />
        <Choice
          selected={data.wwwMode === "addWww"}
          disabled={!data.apexExists}
          primary={data.wwwDomain}
          redirecting={data.apexDomain}
          onSelect={() => choose("addWww")}
        />
        <ChoiceNone selected={data.wwwMode === "none"} onSelect={() => choose("none")} />
      </PopoverContent>
    </Popover>
  );
}

type Role = { kind: "primary" } | { kind: "redirects"; to: string } | { kind: "standalone" };

function roleForRow({
  mode,
  isApexRow,
  sister,
}: {
  mode: WwwMode;
  isApexRow: boolean;
  sister: string;
}): Role {
  if (mode === "none") {
    return { kind: "standalone" };
  }
  const apexIsPrimary = mode === "stripWww";
  const thisIsPrimary = isApexRow ? apexIsPrimary : !apexIsPrimary;
  if (thisIsPrimary) {
    return { kind: "primary" };
  }
  return { kind: "redirects", to: sister };
}

function RoleLabel({ role }: { role: Role }) {
  if (role.kind === "primary") {
    return (
      <>
        <span className="size-1.5 rounded-full bg-success-9" />
        <span>Primary</span>
      </>
    );
  }
  if (role.kind === "redirects") {
    return (
      <>
        <ArrowRight className="size-3! text-gray-11" />
        <span className="font-mono text-[12px]">{role.to}</span>
      </>
    );
  }
  return (
    <>
      <span className="size-1.5 rounded-full bg-gray-7" />
      <span>Standalone</span>
    </>
  );
}

function Choice({
  selected,
  disabled,
  primary,
  redirecting,
  onSelect,
}: {
  selected: boolean;
  disabled: boolean;
  primary: string;
  redirecting: string;
  onSelect: () => void;
}) {
  return (
    <ChoiceShell selected={selected} disabled={disabled} onSelect={onSelect}>
      <div className="font-mono text-[13px] text-gray-12 truncate">{primary}</div>
      <div className="text-[12px] text-gray-9 truncate">
        <span className="font-mono">{redirecting}</span> redirects here
      </div>
    </ChoiceShell>
  );
}

function ChoiceNone({ selected, onSelect }: { selected: boolean; onSelect: () => void }) {
  return (
    <ChoiceShell selected={selected} onSelect={onSelect}>
      <div className="text-[13px] text-gray-12">No primary</div>
      <div className="text-[12px] text-gray-9">both domains serve independently</div>
    </ChoiceShell>
  );
}

function ChoiceShell({
  selected,
  disabled,
  onSelect,
  children,
}: {
  selected: boolean;
  disabled?: boolean;
  onSelect: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onSelect}
      className={cn(
        "grid grid-cols-[16px_minmax(0,1fr)] items-start gap-2 px-2 py-2 rounded-md w-full text-left",
        "hover:bg-gray-3 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent",
      )}
    >
      <div className="mt-1 size-3.5 rounded-full border border-grayA-6 flex items-center justify-center">
        {selected && <span className="size-2 rounded-full bg-accent-9" />}
      </div>
      <div className="min-w-0 space-y-0.5">{children}</div>
    </button>
  );
}
