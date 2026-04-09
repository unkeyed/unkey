"use client";

import { DoubleChevronRight } from "@unkey/icons";
import { SlidePanel } from "@unkey/ui";
import { Children, type ReactNode, isValidElement, useCallback, useRef } from "react";
import {
  type FieldErrors,
  type FieldValues,
  FormProvider,
  type UseFormReturn,
} from "react-hook-form";
import { FormAccordion, type FormAccordionHandle } from "./form-accordion";

type ChildrenOnlyProps = { children: ReactNode };

/** Render-less slot — props are read by PolicyFormRoot via parseSlots. */
function Fields(_props: ChildrenOnlyProps) {
  return null;
}

type AccordionProps = { defaultExpanded?: string; children: ReactNode };
/** Render-less slot — props are read by PolicyFormRoot via parseSlots. */
function Accordion(_props: AccordionProps) {
  return null;
}

const Section = FormAccordion.Section;

/** Render-less slot — props are read by PolicyFormRoot via parseSlots. */
function Footer(_props: ChildrenOnlyProps) {
  return null;
}

type ParsedSlots = {
  fields: ReactNode;
  accordion: AccordionProps | null;
  footer: ReactNode;
};

function parseSlots(children: ReactNode): ParsedSlots {
  const result: ParsedSlots = { fields: null, accordion: null, footer: null };

  Children.forEach(children, (child) => {
    if (!isValidElement(child)) {
      return;
    }
    switch (child.type) {
      case Fields:
        result.fields = (child.props as ChildrenOnlyProps).children;
        break;
      case Accordion:
        result.accordion = child.props as AccordionProps;
        break;
      case Footer:
        result.footer = (child.props as ChildrenOnlyProps).children;
        break;
      default:
        if (process.env.NODE_ENV === "development") {
          console.warn(
            "[PolicyForm] Unknown child — only Fields, Accordion, Footer are valid slots.",
            child,
          );
        }
    }
  });

  return result;
}

type SectionMeta = { id: string; fields?: string[]; catchAll?: boolean };

function parseSectionMeta(accordionChildren: ReactNode): SectionMeta[] {
  const meta: SectionMeta[] = [];
  Children.forEach(accordionChildren, (child) => {
    if (isValidElement(child) && child.type === Section) {
      const { id, fields, catchAll } = child.props as SectionMeta;
      meta.push({ id, fields, catchAll });
    }
  });
  return meta;
}

function findSectionToExpand<T extends FieldValues>(
  errors: FieldErrors<T>,
  sections: SectionMeta[],
): string | null {
  const errorKeys = new Set(Object.keys(errors));
  let catchAllId: string | null = null;

  for (const section of sections) {
    if (section.catchAll) {
      catchAllId = section.id;
    }
    if (section.fields?.some((f) => errorKeys.has(f))) {
      return section.id;
    }
  }

  return errorKeys.size > 0 ? catchAllId : null;
}

type PolicyFormRootProps<T extends FieldValues> = {
  title: string;
  description: React.ReactNode;
  isOpen: boolean;
  topOffset: number;
  onClose: () => void;
  form: UseFormReturn<T>;
  onSubmit: (values: T) => void;
  onInvalid?: (errors: FieldErrors<T>) => void;
  children: ReactNode;
};

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
  const accordionRef = useRef<FormAccordionHandle>(null);
  const slots = parseSlots(children);

  const sectionsRef = useRef<SectionMeta[]>([]);
  sectionsRef.current = slots.accordion ? parseSectionMeta(slots.accordion.children) : [];

  const handleInvalid = useCallback(
    (errors: FieldErrors<T>) => {
      const target = findSectionToExpand(errors, sectionsRef.current);
      if (target) {
        accordionRef.current?.expand(target);
      }
      setTimeout(() => {
        const container = document.querySelector(
          '[aria-invalid="true"], [data-error="true"]',
        ) as HTMLElement | null;
        // For group containers (e.g. method toggles), focus the first interactive child
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
        <FormProvider {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit, handleInvalid)}
            className="h-full flex flex-col"
          >
            <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">
              {slots.fields && <div className="flex flex-col gap-5 px-8">{slots.fields}</div>}

              {slots.accordion && (
                <div className="mt-6 border-b border-gray-4">
                  <FormAccordion
                    ref={accordionRef}
                    defaultExpanded={slots.accordion.defaultExpanded}
                  >
                    {slots.accordion.children}
                  </FormAccordion>
                </div>
              )}
            </div>
            {slots.footer}
          </form>
        </FormProvider>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}

export const PolicyForm = Object.assign(PolicyFormRoot, {
  Fields,
  Accordion,
  Section,
  Footer,
});
