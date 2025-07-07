import { CalendarClock, ChartPie, Code, Gauge, Key2, StackPerspective2 } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { ExpandableSettings } from "../components/expandable-settings";
import type { OnboardingStep } from "../components/onboarding-wizard";

export const useKeyCreationStep = (): OnboardingStep => {
  return {
    name: "API key",
    icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
    body: (
      <form>
        <div className="flex flex-col">
          <FormInput
            placeholder="Enter workspace name"
            label="Workspace name"
            optional
            className="w-full"
          />
          <div className="mt-8" />
          <div className="text-gray-11 text-[13px] leading-6">
            Fine-tune your API key by enabling additional options
          </div>
          <div className="mt-5" />

          <div className="flex flex-col gap-3 w-full">
            <ExpandableSettings
              icon={<Key2 className="text-gray-9 flex-shrink-0" size="sm-regular" />}
              title="General Setup"
              onCheckedChange={(checked) => console.log("General setup:", checked)}
            >
              <div className="text-gray-11 text-[13px] leading-6 mb-3">
                Configure additional settings for your API key
              </div>
              <div className="space-y-2">
                <div className="text-gray-12 text-[13px] leading-6">• Rate limiting options</div>
                <div className="text-gray-12 text-[13px] leading-6">• Expiration settings</div>
                <div className="text-gray-12 text-[13px] leading-6">• Permission scopes</div>
              </div>
            </ExpandableSettings>
            <ExpandableSettings
              icon={<Gauge className="text-gray-9 flex-shrink-0" size="sm-regular" />}
              title="Ratelimit"
              onCheckedChange={(checked) => console.log("Security:", checked)}
            >
              <div className="text-gray-11 text-[13px] leading-6 mb-3">
                Advanced security configurations
              </div>
              <div className="space-y-2">
                <div className="text-gray-12 text-[13px] leading-6">• IP whitelisting</div>
                <div className="text-gray-12 text-[13px] leading-6">• Webhook signatures</div>
              </div>
            </ExpandableSettings>
            <ExpandableSettings
              icon={<ChartPie className="text-gray-9 flex-shrink-0" size="sm-regular" />}
              title="Credits"
              onCheckedChange={(checked) => console.log("Security:", checked)}
            >
              <div className="text-gray-11 text-[13px] leading-6 mb-3">
                Advanced security configurations
              </div>
              <div className="space-y-2">
                <div className="text-gray-12 text-[13px] leading-6">• IP whitelisting</div>
                <div className="text-gray-12 text-[13px] leading-6">• Webhook signatures</div>
              </div>
            </ExpandableSettings>
            <ExpandableSettings
              icon={<CalendarClock className="text-gray-9 flex-shrink-0" size="sm-regular" />}
              title="Expiration"
              onCheckedChange={(checked) => console.log("Security:", checked)}
            >
              <div className="text-gray-11 text-[13px] leading-6 mb-3">
                Advanced security configurations
              </div>
              <div className="space-y-2">
                <div className="text-gray-12 text-[13px] leading-6">• IP whitelisting</div>
                <div className="text-gray-12 text-[13px] leading-6">• Webhook signatures</div>
              </div>
            </ExpandableSettings>
            <ExpandableSettings
              icon={<Code className="text-gray-9 flex-shrink-0" size="sm-regular" />}
              title="Metadata"
              onCheckedChange={(checked) => console.log("Security:", checked)}
            >
              <div className="text-gray-11 text-[13px] leading-6 mb-3">
                Advanced security configurations
              </div>
              <div className="space-y-2">
                <div className="text-gray-12 text-[13px] leading-6">• IP whitelisting</div>
                <div className="text-gray-12 text-[13px] leading-6">• Webhook signatures</div>
              </div>
            </ExpandableSettings>
          </div>
        </div>
      </form>
    ),
    kind: "non-required" as const,
    buttonText: "Continue",
    description: "Setup your API key with extended configurations",
    onStepNext: () => {
      console.info("Going next from workspace step");
    },
    onStepBack: () => {
      console.info("Going back from workspace step");
    },
  };
};
