"use client";

import * as React from "react";

import { Accordion, AccordionContent, AccordionItem } from "@/components/ui/accordion";
import { Switch } from "@/components/ui/switch";
import { type AllConsentNames, useConsentManager } from "@koroflow/core-react";
import { ChevronDown } from "lucide-react";
import { PrimaryButton, SecondaryButton } from "../button";
import { usePostHog } from "posthog-js/react";

interface ConsentCustomizationWidgetProps extends React.HTMLAttributes<HTMLDivElement> {
  onSave?: () => void;
}

const ConsentCustomizationWidget = React.forwardRef<
  HTMLDivElement,
  ConsentCustomizationWidgetProps
>(({ onSave, ...props }, ref) => {
  const posthog = usePostHog();

  const { consents, setConsent, saveConsents, getDisplayedConsents } = useConsentManager();
  const [openItems, setOpenItems] = React.useState<string[]>([]);

  const toggleAccordion = React.useCallback((value: string) => {
    setOpenItems((prev) =>
      prev.includes(value) ? prev.filter((item) => item !== value) : [...prev, value],
    );
  }, []);

  const handleSaveConsents = React.useCallback(() => {
    saveConsents("custom");
    posthog.capture("consent", {
      consent: "custom",
	  consents,
    });

    if (onSave) {
      onSave();
    }
  }, [saveConsents, posthog, consents, onSave]);

  const handleConsentChange = React.useCallback(
    (name: AllConsentNames, checked: boolean) => {
      setConsent(name, checked);
    },
    [setConsent],
  );

  const acceptAll = React.useCallback(() => {
    const allConsents = Object.keys(consents) as (keyof typeof consents)[];
    for (const consentName of allConsents) {
      setConsent(consentName, true);
    }
    saveConsents("all");
    posthog.capture("consent", {
      consent: "all",
    });
    if (onSave) {
      onSave();
    }
  }, [consents, setConsent, posthog, onSave, saveConsents]);

  const rejectAll = React.useCallback(() => {
    saveConsents("necessary");
    posthog.capture("consent", {
      consent: "necessary",
    });
    if (onSave) {
      onSave();
    }
  }, [saveConsents, posthog, onSave]);

  return (
    <div className="space-y-6 pt-8" ref={ref} {...props}>
      <Accordion
        type="multiple"
        value={openItems}
        onValueChange={setOpenItems}
        className="w-full border border-white/20 rounded-lg"
      >
        {getDisplayedConsents().map((consent) => (
          <AccordionItem
            value={consent.name}
            key={consent.name}
            className="px-4 [&:not(:last-child)]:border-b [&:not(:last-child)]:border-b-white/20"
          >
            <div className="flex items-center justify-between py-4">
              <div
                className="flex-grow"
                role="button"
                tabIndex={0}
                onClick={() => toggleAccordion(consent.name)}
                onKeyUp={(e) => e.key === "Enter" && toggleAccordion(consent.name)}
                onKeyDown={(e) => e.key === "Enter" && toggleAccordion(consent.name)}
              >
                <div className="flex items-center justify-between cursor-pointer">
                  <span className="font-medium capitalize">{consent.name.replace("_", " ")}</span>
                  <ChevronDown
                    className={`h-4 w-4 shrink-0 transition-transform duration-200 ${
                      openItems.includes(consent.name) ? "rotate-180" : ""
                    }`}
                  />
                </div>
              </div>
              <Switch
                checked={consents[consent.name]}
                onCheckedChange={(checked) => handleConsentChange(consent.name, checked)}
                disabled={consent.disabled}
                className="ml-4"
              />
            </div>
            <AccordionContent>
              <p className="text-sm text-muted-foreground pb-4">{consent.description}</p>
            </AccordionContent>
          </AccordionItem>
        ))}
      </Accordion>
      <div className="flex justify-between">
        <div className="flex justify-between space-x-2">
          <SecondaryButton
            onClick={rejectAll}
            label="Deny"
            className="w-full text-sm sm:w-auto cursor-pointer"
          />
          <SecondaryButton
            onClick={acceptAll}
            label="Accept All"
            className="w-full text-sm sm:w-auto cursor-pointer"
          />
        </div>
        <PrimaryButton onClick={handleSaveConsents} label="Save" className="cursor-pointer" />
      </div>
    </div>
  );
});

ConsentCustomizationWidget.displayName = "ConsentCustomizationWidget";

export default ConsentCustomizationWidget;
