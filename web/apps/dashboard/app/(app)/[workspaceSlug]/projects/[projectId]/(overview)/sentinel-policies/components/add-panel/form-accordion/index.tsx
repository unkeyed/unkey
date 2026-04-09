"use client";

import {
  Children,
  type ReactElement,
  type ReactNode,
  forwardRef,
  isValidElement,
  useCallback,
  useImperativeHandle,
  useState,
} from "react";
import { AccordionSection } from "./section";

type SectionProps = {
  id: string;
  label: string;
  summary: ReactNode;
  children: ReactNode;
  tooltipContent?: ReactNode;
  /** Shown in the header only when the section is collapsed. */
  collapsedAction?: ReactNode;
  /** Form field names whose errors should trigger auto-expansion of this section. */
  fields?: string[];
  /** Expand this section when any error isn't claimed by another section's `fields`. */
  catchAll?: boolean;
};

/**
 * Render-less marker component. `FormAccordion` reads its props to
 * build the accordion UI -- the component itself renders nothing.
 */
function Section(_props: SectionProps): ReactElement | null {
  return null;
}

export type FormAccordionHandle = {
  expand: (id: string) => void;
};

type FormAccordionProps = {
  defaultExpanded?: string;
  children: ReactNode;
};

const FormAccordionRoot = forwardRef<FormAccordionHandle, FormAccordionProps>(
  ({ defaultExpanded, children }, ref) => {
    const [expanded, setExpanded] = useState<string | null>(defaultExpanded ?? null);

    const toggle = useCallback(
      (id: string) => setExpanded((prev) => (prev === id ? null : id)),
      [],
    );

    useImperativeHandle(ref, () => ({ expand: setExpanded }), []);

    const sections: SectionProps[] = [];
    Children.forEach(children, (child) => {
      if (isValidElement(child) && child.type === Section) {
        sections.push(child.props as SectionProps);
      }
    });

    return (
      <>
        {sections.map((s) => {
          const isActive = s.id === expanded;
          return (
            <AccordionSection
              key={s.id}
              label={s.label}
              summary={s.summary}
              active={isActive}
              onToggle={() => toggle(s.id)}
              tooltipContent={s.tooltipContent}
              headerAction={isActive ? undefined : s.collapsedAction}
            >
              {isActive ? s.children : null}
            </AccordionSection>
          );
        })}
      </>
    );
  },
);

FormAccordionRoot.displayName = "FormAccordion";

export const FormAccordion = Object.assign(FormAccordionRoot, { Section });
