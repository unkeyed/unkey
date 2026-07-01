"use client";

import { trpc } from "@/lib/trpc/client";
import { Clone, ShieldKey, TriangleWarning } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import type React from "react";
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

function ShareCard({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-6 rounded-2xl border border-gray-5 p-8 text-center">
      {children}
    </div>
  );
}

function CardGlyph({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex size-12 items-center justify-center rounded-xl border border-gray-5 bg-grayA-2 text-gray-12">
      {children}
    </div>
  );
}

function CardText({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex flex-col items-center gap-1">
      <h1 className="font-semibold text-gray-12 text-[16px] leading-[24px]">{title}</h1>
      <p className="text-gray-10 text-[13px] leading-[20px]">{description}</p>
    </div>
  );
}

export function ShareReveal() {
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
      <ShareCard>
        <CardGlyph>
          <TriangleWarning iconSize="xl-medium" aria-hidden="true" />
        </CardGlyph>
        <CardText
          title="This link has expired or was already used"
          description="Secret links open only once. Ask the sender to send you a new one."
        />
      </ShareCard>
    );
  }

  if (state.status === "revealed") {
    return (
      <ShareCard>
        <CardGlyph>
          <ShieldKey iconSize="xl-medium" aria-hidden="true" />
        </CardGlyph>
        <CardText
          title="Your key"
          description="Copy your key and keep it safe, this link will no longer work once you close this window."
        />
        <div className="flex w-full items-center rounded-xl border border-gray-5 bg-grayA-2 px-3 py-2 focus-within:ring-2 focus-within:ring-gray-6">
          <input
            readOnly
            value={state.key}
            aria-label="Shared key"
            onFocus={(e) => e.currentTarget.select()}
            className="flex-1 truncate bg-transparent font-mono text-[13px] text-gray-12 outline-none"
          />
        </div>
        <Button
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(state.key);
              toast.success("Key copied");
            } catch {
              toast.error("Couldn't copy. Select the key and copy it manually.");
            }
          }}
        >
          <Clone iconSize="sm-regular" />
          Copy key
        </Button>
      </ShareCard>
    );
  }

  // ready | revealing
  return (
    <ShareCard>
      <CardGlyph>
        <ShieldKey iconSize="xl-medium" aria-hidden="true" />
      </CardGlyph>
      <CardText
        title="You've been sent a secure key"
        description="This link is one-time. Reveal it when you're ready to copy, it can't be opened again."
      />
      <Button
        variant="primary"
        size="xlg"
        className="w-full rounded-lg"
        onClick={onReveal}
        loading={state.status === "revealing"}
        disabled={state.status === "revealing"}
      >
        Reveal key
      </Button>
    </ShareCard>
  );
}
