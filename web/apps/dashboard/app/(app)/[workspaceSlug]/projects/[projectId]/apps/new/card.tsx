"use client";

import { Card, Loading, cn } from "@unkey/ui";
import { type ReactNode, type Ref, useEffect, useState } from "react";

const MIN_HIDDEN_PX = 8;

type SurfaceProps = {
  className?: string;
  children?: ReactNode;
};

export function CardSurface({ className, children }: SurfaceProps) {
  return <Card className={cn("flex min-h-0 flex-col overflow-hidden", className)}>{children}</Card>;
}

export function CardFrame({
  copy,
  eyebrow,
  back,
  actionsRef,
  footerHidden = false,
  children,
}: {
  copy: { title: string; description: string };
  eyebrow?: string;
  back: ReactNode;
  actionsRef?: Ref<HTMLDivElement>;
  footerHidden?: boolean;
  children: ReactNode;
}) {
  return (
    <CardSurface className="@container">
      <CardScrollBody>
        <div className="shrink-0">
          <CardTitle eyebrow={eyebrow} title={copy.title} description={copy.description} />
        </div>
        {children}
      </CardScrollBody>
      <CardFooter hidden={footerHidden}>
        {back}
        <div ref={actionsRef} className="ml-auto flex items-center gap-3" />
      </CardFooter>
    </CardSurface>
  );
}

function CardFooter({ hidden = false, children }: { hidden?: boolean; children: ReactNode }) {
  return (
    <div
      className={cn(
        "flex shrink-0 items-center gap-3 border-t bg-grayA-2 px-5 py-3",
        hidden && "hidden",
      )}
    >
      {children}
    </div>
  );
}

function CardTitle({
  eyebrow,
  title,
  description,
}: {
  eyebrow?: string;
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col gap-1">
      {eyebrow ? (
        <span className="hidden text-xs text-gray-11 lg:max-xl:block">{eyebrow}</span>
      ) : null}
      <h1 tabIndex={-1} className="text-lg font-semibold leading-7 text-gray-12 outline-hidden">
        {title}
      </h1>
      <p className="text-sm leading-5 text-gray-11">{description}</p>
    </div>
  );
}

export function CardPending({
  message,
  description,
  className,
}: {
  message: string;
  description?: string;
  className?: string;
}) {
  return (
    <div
      aria-live="polite"
      className={cn(
        "flex h-36 flex-col items-center justify-center gap-3 text-sm text-gray-11",
        className,
      )}
    >
      <Loading size={18} />
      <div className="flex flex-col items-center gap-0.5">
        <p>{message}</p>
        {description ? <p className="text-xs text-gray-10">{description}</p> : null}
      </div>
    </div>
  );
}

function useScrollFade() {
  const [element, setElement] = useState<HTMLElement | null>(null);
  useEffect(() => {
    if (!element) {
      return;
    }
    const update = () => {
      const below = element.scrollHeight - element.scrollTop - element.clientHeight > MIN_HIDDEN_PX;
      element.toggleAttribute("data-more-below", below);
    };
    update();
    const resize = new ResizeObserver(update);
    resize.observe(element);
    for (const child of element.children) {
      resize.observe(child);
    }
    const mutations = new MutationObserver(update);
    mutations.observe(element, { childList: true, subtree: true });
    element.addEventListener("scroll", update, { passive: true });
    return () => {
      resize.disconnect();
      mutations.disconnect();
      element.removeEventListener("scroll", update);
    };
  }, [element]);
  return setElement;
}

function CardScrollBody({ children }: { children: ReactNode }) {
  const scrollRef = useScrollFade();
  return (
    <div
      ref={scrollRef}
      className="flex min-h-0 flex-col gap-5 overflow-y-auto overscroll-contain p-5 [scrollbar-width:thin] data-[more-below]:[mask-image:linear-gradient(to_bottom,black_calc(100%-40px),transparent)]"
    >
      {children}
    </div>
  );
}
