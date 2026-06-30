"use client";

import { SecretKey } from "@/components/secret-key";
import { trpc } from "@/lib/trpc/client";
import { ShieldKey, TriangleWarning } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";

type State =
  | { status: "loading" }
  | { status: "ready"; id: string }
  | { status: "revealing"; id: string }
  | { status: "revealed"; key: string }
  | { status: "error" };

function parseShareId(hash: string): string | null {
  const id = hash.startsWith("#") ? hash.slice(1) : hash;
  return id.length > 0 ? id : null;
}

export const ShareReveal = () => {
  const [state, setState] = useState<State>({ status: "loading" });
  const { mutateAsync: reveal } = trpc.share.reveal.useMutation();
  // Reveal is one-time, so guard against a double-fire (fast double-click).
  const revealing = useRef(false);

  useEffect(() => {
    // Only read the fragment; the reveal is gated behind an explicit click so an
    // accidental open, a link preview, or a JS-executing scanner can't burn it.
    const id = parseShareId(window.location.hash);
    setState(id ? { status: "ready", id } : { status: "error" });
  }, []);

  const onReveal = async () => {
    if (revealing.current || state.status !== "ready") {
      return;
    }
    revealing.current = true;
    const { id } = state;
    setState({ status: "revealing", id });

    try {
      const result = await reveal({ id });
      if (!result.ok) {
        setState({ status: "error" });
        return;
      }
      // Strip the id from the URL only after a successful reveal, so a transient
      // failure can still be retried by reloading.
      window.history.replaceState(null, "", window.location.pathname + window.location.search);
      setState({ status: "revealed", key: result.secret });
    } catch {
      setState({ status: "error" });
    }
  };

  if (state.status === "loading") {
    return null;
  }

  if (state.status === "error") {
    return (
      <Empty>
        <Empty.Icon>
          <TriangleWarning />
        </Empty.Icon>
        <Empty.Title>This link can't be opened</Empty.Title>
        <Empty.Description>
          It may have already been viewed or expired. Links open only once. Ask the sender for a new
          one.
        </Empty.Description>
      </Empty>
    );
  }

  if (state.status === "revealed") {
    return (
      <div className="flex flex-col items-center gap-4 w-full max-w-[560px]">
        <ShieldKey iconSize="2xl-thin" className="text-gray-12" aria-hidden="true" />
        <div className="flex flex-col gap-1 items-center text-center">
          <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
            Here is your key
          </div>
          <div className="text-gray-10 text-[13px] leading-[20px]">
            Copy it now. This link has been used and won't work again.
          </div>
        </div>
        <SecretKey value={state.key} title="Shared key" className="w-full" />
      </div>
    );
  }

  // ready | revealing
  return (
    <Empty>
      <Empty.Icon>
        <ShieldKey />
      </Empty.Icon>
      <Empty.Title>You've been sent a key</Empty.Title>
      <Empty.Description>
        Reveal it only when you're ready to copy it. The link works once and stops working
        afterwards.
      </Empty.Description>
      <Empty.Actions>
        <Button
          variant="primary"
          onClick={onReveal}
          loading={state.status === "revealing"}
          disabled={state.status === "revealing"}
        >
          Reveal key
        </Button>
      </Empty.Actions>
    </Empty>
  );
};
