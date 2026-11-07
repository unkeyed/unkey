"use client";
import { Toolbar } from "@base-ui/react/toolbar";
import { IconXmarkOutline18 } from "@unkey/icons";
import {
  type FocusEvent,
  type ReactNode,
  type RefObject,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { Button, type ButtonProps } from "./buttons/button";
import { usePageElement } from "./page/page-container";
import { Tooltip, TooltipContent, TooltipTrigger } from "./tooltip";

type SelectionBarProps = {
  count: number;
  onClear: () => void;
  detail?: ReactNode;
  children: ReactNode;
};

export function SelectionBar({ count, onClear, detail, children }: SelectionBarProps) {
  return (
    <>
      <output aria-live="polite" className="sr-only">
        {count > 0 ? `${count} selected` : ""}
      </output>
      {count > 0 ? (
        <SelectionBarPill count={count} onClear={onClear} detail={detail}>
          {children}
        </SelectionBarPill>
      ) : null}
    </>
  );
}

function SelectionBarPill({ count, onClear, detail, children }: SelectionBarProps) {
  const barRef = useRef<HTMLDivElement>(null);
  const page = usePageElement("SelectionBar");
  const frame = usePageFrame(page);
  const rememberReturnFocus = useReturnFocusOnUnmount(barRef, page);
  useClearOnEscape(onClear);

  return (
    <div
      ref={barRef}
      data-slot="selection-bar"
      style={frame}
      onFocus={rememberReturnFocus}
      className="@container pointer-events-none fixed right-0 bottom-5 left-0 z-10 flex justify-center px-4"
    >
      <Toolbar.Root
        aria-label="Selection actions"
        className="pointer-events-auto flex h-11 min-w-0 max-w-full animate-fade-slide-in-fast items-center gap-2 rounded-surface bg-raised pr-1.5 pl-3 shadow-floating motion-reduce:animate-none"
      >
        <span className="flex h-4.5 min-w-4.5 shrink-0 items-center justify-center rounded-md bg-gray-12 px-1 font-medium text-2xs text-white tabular-nums ring-2 ring-gray-6 dark:text-black">
          {count}
        </span>
        <span className="shrink-0 text-gray-11 text-sm">selected</span>
        {detail}
        <Toolbar.Separator orientation="vertical" className="h-5 w-px shrink-0 bg-grayA-4" />
        <Toolbar.Group className="flex items-center gap-1.5">{children}</Toolbar.Group>
        <Toolbar.Button
          render={<Button size="icon" variant="ghost" className="shrink-0 [&_svg]:size-3.5" />}
          onClick={onClear}
          aria-label="Clear selection"
        >
          <IconXmarkOutline18 />
        </Toolbar.Button>
      </Toolbar.Root>
    </div>
  );
}

type SelectionBarActionProps = Omit<
  ButtonProps,
  "variant" | "size" | "color" | "render" | "keyboard" | "children"
> & {
  icon: ReactNode;
  label: string;
};

export function SelectionBarAction({
  icon,
  label,
  loading,
  disabled,
  className,
  ...props
}: SelectionBarActionProps) {
  const labelRef = useRef<HTMLSpanElement>(null);
  const [tooltipOpen, setTooltipOpen] = useState(false);

  return (
    <Tooltip
      open={tooltipOpen}
      onOpenChange={(open) => setTooltipOpen(open && !labelRef.current?.checkVisibility())}
    >
      <TooltipTrigger
        render={
          <Toolbar.Button
            disabled={disabled}
            aria-label={label}
            render={
              <Button
                variant="outline"
                size="sm"
                loading={loading}
                focusableWhenDisabled
                className={className}
              />
            }
            {...props}
          />
        }
      >
        {icon}
        <span ref={labelRef} className="@max-3xl:hidden">
          {label}
        </span>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

const TEXT_ENTRY_SELECTOR =
  'textarea, select, input:not([type="checkbox"], [type="radio"], [type="button"], [type="submit"], [type="reset"], [type="range"], [type="color"], [type="file"])';

function isEscapeOwnedElsewhere(event: KeyboardEvent) {
  const target = event.target;
  if (
    target instanceof HTMLElement &&
    (target.isContentEditable || target.closest(TEXT_ENTRY_SELECTOR))
  ) {
    return true;
  }
  return (
    document.querySelector('[role="dialog"]:not([hidden]), [role="alertdialog"]:not([hidden])') !==
    null
  );
}

function useClearOnEscape(onClear: () => void) {
  const onClearRef = useRef(onClear);
  useLayoutEffect(() => {
    onClearRef.current = onClear;
  });

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.repeat || isEscapeOwnedElsewhere(event)) {
        return;
      }
      clearTimeout(timer);
      // Base UI popups dismiss on Escape from their own document listener, which may run after this one, and call preventDefault.
      timer = setTimeout(() => {
        if (!event.defaultPrevented) {
          onClearRef.current();
        }
      });
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      clearTimeout(timer);
    };
  }, []);
}

function useReturnFocusOnUnmount(
  barRef: RefObject<HTMLDivElement | null>,
  page: HTMLElement | null,
) {
  const returnFocusRef = useRef<HTMLElement | null>(null);

  useLayoutEffect(() => {
    const bar = barRef.current;
    return () => {
      const target = returnFocusRef.current;
      if (!target || !bar?.contains(document.activeElement)) {
        return;
      }
      requestAnimationFrame(() => {
        if (target.isConnected) {
          target.focus({ preventScroll: true });
        } else if (page && document.activeElement === document.body) {
          focusTemporarily(page);
        }
      });
    };
  }, [barRef, page]);

  return (event: FocusEvent<HTMLDivElement>) => {
    const from = event.relatedTarget;
    if (!(from instanceof HTMLElement) || event.currentTarget.contains(from)) {
      return;
    }
    if (from.closest('[role="dialog"]') && returnFocusRef.current?.isConnected) {
      return;
    }
    returnFocusRef.current = from;
  };
}

function focusTemporarily(element: HTMLElement) {
  if (!element.hasAttribute("tabindex")) {
    element.tabIndex = -1;
    element.addEventListener("blur", () => element.removeAttribute("tabindex"), { once: true });
  }
  element.focus({ preventScroll: true });
}

type PageFrame = { left: number; width: number };

function usePageFrame(page: HTMLElement | null): PageFrame | undefined {
  const [frame, setFrame] = useState<PageFrame>();
  useLayoutEffect(() => {
    if (!page) {
      return;
    }
    const measure = () => {
      const { left, width } = page.getBoundingClientRect();
      setFrame((current) =>
        current?.left === left && current.width === width ? current : { left, width },
      );
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(page);
    return () => observer.disconnect();
  }, [page]);
  return frame;
}
