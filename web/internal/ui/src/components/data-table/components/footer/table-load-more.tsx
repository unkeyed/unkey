"use client";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { Button } from "../../../buttons/button";
import { loadedCountLabel } from "./loaded-count";
import { FOOTER_ERROR_COPY, type FooterKind, type ListFooterState } from "./types";

export type TableLoadMoreProps = {
  state: ListFooterState;
  totalCount: number | null;
  singular: string;
  plural: string;
  scope?: string;
};

type FooterStates = { [S in ListFooterState as S["kind"]]: S };

type FooterButton = {
  label: string;
  onClick: (() => void) | undefined;
  inert: boolean;
  retrying: boolean;
  alert: string | null;
};

const FOOTER_BUTTONS: {
  [K in FooterKind]: (state: FooterStates[K], plural: string) => FooterButton | null;
} = {
  hidden: () => null,
  more: (state) => ({
    label: "Load more",
    onClick: state.onLoadMore,
    inert: false,
    retrying: false,
    alert: null,
  }),
  loadingMore: () => ({
    label: "Load more",
    onClick: undefined,
    inert: true,
    retrying: false,
    alert: null,
  }),
  error: (state, plural) => ({
    label: "Retry",
    onClick: state.onRetry,
    inert: false,
    retrying: state.retrying,
    alert: FOOTER_ERROR_COPY[state.reason](plural),
  }),
};

function footerButtonOf<K extends FooterKind>(kind: K, state: FooterStates[K], plural: string) {
  return FOOTER_BUTTONS[kind](state, plural);
}

export function TableLoadMore({
  state,
  loadedCount,
  ...count
}: TableLoadMoreProps & { loadedCount: number }) {
  const sentinelRef = useAutoLoad(state.kind === "more" ? state.onLoadMore : null);
  const isSlowLoad = useSlowFlag(state.kind === "loadingMore");
  const label = `Showing ${loadedCountLabel({ loadedCount, ...count })}`;
  const announcement = useLoadedAnnouncement(state.kind, label, loadedCount);
  const button = footerButtonOf(state.kind, state, count.plural);
  const endFocus = useEndFocus(state.kind);
  const progress = isSlowLoad ? `Loading more ${count.plural}…` : label;

  return (
    <div>
      <span className="sr-only" aria-live="polite">
        {isSlowLoad ? progress : announcement}
      </span>
      {button ? (
        <div
          ref={sentinelRef}
          onFocus={endFocus.onFocus}
          onBlur={endFocus.onBlur}
          className="flex flex-col items-center gap-1.5 py-4"
        >
          <Button
            variant="outline"
            size="sm"
            aria-disabled={button.inert || undefined}
            loading={button.retrying}
            onClick={button.onClick}
          >
            {button.label}
          </Button>
          {button.alert ? (
            <span role="alert" className="text-xs text-error-11">
              {button.alert}
            </span>
          ) : (
            <span className="text-xs tabular-nums text-grayA-11">{progress}</span>
          )}
        </div>
      ) : loadedCount > 0 ? (
        <div
          ref={endFocus.endRef}
          tabIndex={-1}
          className="py-4 text-center text-xs tabular-nums text-grayA-11 outline-hidden"
        >
          {label}
        </div>
      ) : null}
    </div>
  );
}

function useAutoLoad(onLoadMore: (() => void) | null) {
  const pendingRef = useRef(onLoadMore);
  const observerRef = useRef<{ observer: IntersectionObserver; node: Element } | null>(null);
  const armed = onLoadMore !== null;

  useLayoutEffect(() => {
    pendingRef.current = onLoadMore;
  });

  useEffect(() => {
    const current = observerRef.current;
    if (armed && current) {
      current.observer.unobserve(current.node);
      current.observer.observe(current.node);
    }
  }, [armed]);

  return useCallback((node: HTMLDivElement | null) => {
    if (!node || typeof IntersectionObserver === "undefined") {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        const load = pendingRef.current;
        if (load && entries.some((entry) => entry.isIntersecting)) {
          pendingRef.current = null;
          load();
        }
      },
      { rootMargin: "50% 0px" },
    );
    observer.observe(node);
    observerRef.current = { observer, node };
    return () => {
      observer.disconnect();
      observerRef.current = null;
    };
  }, []);
}

function useEndFocus(kind: FooterKind) {
  const endRef = useRef<HTMLDivElement>(null);
  const buttonFocusedRef = useRef(false);
  const kindRef = useRef(kind);

  useLayoutEffect(() => {
    const lastPageLoaded = kindRef.current === "loadingMore" && kind === "hidden";
    kindRef.current = kind;
    if (lastPageLoaded && buttonFocusedRef.current) {
      endRef.current?.focus();
    }
    if (kind === "hidden") {
      buttonFocusedRef.current = false;
    }
  }, [kind]);

  return {
    endRef,
    onFocus: () => {
      buttonFocusedRef.current = true;
    },
    onBlur: () => {
      buttonFocusedRef.current = false;
    },
  };
}

function useSlowFlag(active: boolean) {
  const [isSlow, setIsSlow] = useState(false);
  useEffect(() => {
    if (!active) {
      setIsSlow(false);
      return;
    }
    const timer = setTimeout(() => setIsSlow(true), 300);
    return () => clearTimeout(timer);
  }, [active]);
  return isSlow;
}

function useLoadedAnnouncement(kind: FooterKind, label: string, loadedCount: number) {
  const [announced, setAnnounced] = useState({ kind, text: "" });
  if (announced.kind !== kind) {
    const finishedLoad =
      announced.kind === "loadingMore" &&
      (kind === "more" || (kind === "hidden" && loadedCount > 0));
    setAnnounced({ kind, text: finishedLoad ? label : announced.text });
  }
  return announced.text;
}
