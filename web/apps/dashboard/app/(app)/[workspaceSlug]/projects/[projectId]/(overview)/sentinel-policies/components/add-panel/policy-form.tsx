"use client";

import { ChevronDown, CircleInfo, DoubleChevronRight } from "@unkey/icons";
import { InfoTooltip, SlidePanel } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  Children,
  type ReactNode,
  createContext,
  isValidElement,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  type FieldErrors,
  type FieldValues,
  FormProvider,
  type UseFormReturn,
} from "react-hook-form";

type PolicyFormRootProps<T extends FieldValues> = {
  title: string;
  description: ReactNode;
  isOpen: boolean;
  topOffset: number;
  onClose: () => void;
  form: UseFormReturn<T>;
  onSubmit: (values: T) => void;
  onInvalid?: (errors: FieldErrors<T>) => void;
  children: ReactNode;
};

/**
 * Compound component for sentinel policy slide-panel forms.
 *
 * Children are split at render time: `Footer` is pulled out and pinned
 * below the scrollable body so it stays visible regardless of content height.
 * Accordion state is shared via context so `handleInvalid` can auto-expand
 * the section that owns the first validation error.
 */
function PolicyFormRoot<T extends FieldValues>({
  title,
  description,
  isOpen,
  topOffset,
  onClose,
  form,
  onSubmit,
  onInvalid,
  children,
}: PolicyFormRootProps<T>) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const sectionsRef = useRef<Map<string, SectionMeta>>(new Map());

  const toggle = useCallback((id: string) => setExpanded((prev) => (prev === id ? null : id)), []);

  const registerSection = useCallback((meta: SectionMeta) => {
    sectionsRef.current.set(meta.id, meta);
    return () => {
      sectionsRef.current.delete(meta.id);
    };
  }, []);

  const ctxValue: PolicyFormContextValue = { expanded, toggle, registerSection };
  const { body, footer } = splitFooter(children);

  const handleInvalid = useCallback(
    (errors: FieldErrors<T>) => {
      const errorKeys = new Set(Object.keys(errors));
      let catchAllId: string | null = null;
      let targetId: string | null = null;

      for (const section of sectionsRef.current.values()) {
        if (section.catchAll) {
          catchAllId = section.id;
        }
        if (section.fields?.some((f) => errorKeys.has(f))) {
          targetId = section.id;
          break;
        }
      }

      const target = targetId ?? (errorKeys.size > 0 ? catchAllId : null);
      if (target) {
        setExpanded(target);
      }

      setTimeout(() => {
        const container = document.querySelector(
          '[aria-invalid="true"], [data-error="true"]',
        ) as HTMLElement | null;
        const el =
          (container?.matches("input, button, select, textarea") ?? true)
            ? container
            : ((container?.querySelector(
                "button, input, select, textarea",
              ) as HTMLElement | null) ?? container);
        el?.focus();
        el?.scrollIntoView({ behavior: "smooth", block: "center" });
      }, 0);

      onInvalid?.(errors);
    },
    [onInvalid],
  );

  return (
    <SlidePanel.Root isOpen={isOpen} onClose={onClose} topOffset={topOffset}>
      <SlidePanel.Header>
        <div className="flex flex-col">
          <span className="text-gray-12 font-medium text-base leading-8">{title}</span>
          <span className="text-gray-11 text-[13px] leading-5">{description}</span>
        </div>
        <SlidePanel.Close
          aria-label="Close panel"
          className="mt-0.5 inline-flex items-center justify-center size-9 rounded-md hover:bg-grayA-3 transition-colors cursor-pointer"
        >
          <DoubleChevronRight
            iconSize="lg-medium"
            className="text-gray-10 transition-transform duration-300 ease-out group-hover:text-gray-12"
          />
        </SlidePanel.Close>
      </SlidePanel.Header>

      <SlidePanel.Content>
        <PolicyFormContext.Provider value={ctxValue}>
          <FormProvider {...form}>
            <form
              onSubmit={form.handleSubmit(onSubmit, handleInvalid)}
              className="h-full flex flex-col"
            >
              <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">{body}</div>
              {footer}
            </form>
          </FormProvider>
        </PolicyFormContext.Provider>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}

function Fields({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-5 px-8">{children}</div>;
}

function Accordion({
  defaultExpanded,
  children,
}: {
  defaultExpanded?: string;
  children: ReactNode;
}) {
  const { expanded, toggle } = usePolicyForm();
  const initialized = useRef(false);

  useEffect(() => {
    if (!initialized.current && defaultExpanded && expanded === null) {
      toggle(defaultExpanded);
      initialized.current = true;
    }
  }, [defaultExpanded, expanded, toggle]);

  return <div className="mt-6 border-b border-gray-4">{children}</div>;
}

type SectionProps = {
  id: string;
  label: string;
  summary: ReactNode;
  children: ReactNode;
  tooltipContent?: ReactNode;
  collapsedAction?: ReactNode;
  /** Errors on these fields auto-expand this section. */
  fields?: string[];
  /** Fallback section to expand when no other section claims the error field. */
  catchAll?: boolean;
};

function Section({
  id,
  label,
  summary,
  children,
  tooltipContent,
  collapsedAction,
  fields,
  catchAll,
}: SectionProps) {
  const { expanded, toggle, registerSection } = usePolicyForm();
  const isActive = expanded === id;

  useEffect(() => {
    return registerSection({ id, fields, catchAll });
  }, [id, fields, catchAll, registerSection]);

  return (
    <div className="border-t border-grayA-4">
      <div className="flex items-center hover:bg-grayA-2 transition-colors">
        <button
          type="button"
          onClick={() => toggle(id)}
          className="flex-1 min-w-0 px-8 py-3 flex items-center justify-between gap-4 cursor-pointer"
        >
          <span className="flex items-center gap-2 text-[13px] text-gray-11 font-medium">
            <ChevronDown
              iconSize="sm-regular"
              className={cn("transition-transform duration-200", isActive ? "" : "-rotate-90")}
            />
            {label}
            {tooltipContent && (
              <InfoTooltip content={tooltipContent} asChild>
                <span
                  className="ml-0.5 inline-flex items-center text-gray-9 hover:text-gray-11"
                  onClick={(e) => e.stopPropagation()}
                  onKeyDown={(e) => e.stopPropagation()}
                >
                  <CircleInfo iconSize="md-medium" aria-hidden="true" />
                  <span className="sr-only">More info</span>
                </span>
              </InfoTooltip>
            )}
          </span>
          <span className="text-[12px] text-gray-11 truncate">{summary}</span>
        </button>
        {!isActive && collapsedAction && <div className="pr-8 shrink-0">{collapsedAction}</div>}
      </div>
      {isActive && <div className="px-8 pb-6 pt-3">{children}</div>}
    </div>
  );
}

function Footer({ children }: { children: ReactNode }) {
  return <>{children}</>;
}

/** Separates Footer from other children so it renders outside the scrollable area. */
function splitFooter(children: ReactNode): { body: ReactNode[]; footer: ReactNode } {
  const body: ReactNode[] = [];
  let footer: ReactNode = null;

  Children.forEach(children, (child) => {
    if (isValidElement(child) && child.type === Footer) {
      footer = child;
    } else {
      body.push(child);
    }
  });

  return { body, footer };
}

function usePolicyForm() {
  const ctx = useContext(PolicyFormContext);
  if (!ctx) {
    throw new Error("PolicyForm compound components must be rendered inside <PolicyForm>");
  }
  return ctx;
}

type SectionMeta = { id: string; fields?: string[]; catchAll?: boolean };

type PolicyFormContextValue = {
  expanded: string | null;
  toggle: (id: string) => void;
  registerSection: (meta: SectionMeta) => () => void;
};

const PolicyFormContext = createContext<PolicyFormContextValue | null>(null);

export const PolicyForm = Object.assign(PolicyFormRoot, {
  Fields,
  Accordion,
  Section,
  Footer,
});
