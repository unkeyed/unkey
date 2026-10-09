"use client";

import {
  IconCubeOutline18,
  IconHammer2Outline18,
  IconLinkOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import {
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { Fragment } from "react";

const steps = [
  {
    label: "Connect",
    Icon: IconLinkOutline18,
    title: "Grant one-way access",
    text: "Choose an app in the same project. A connection from web to api lets web call api over the private network. It does not let api call web.",
  },
  {
    label: "Target",
    Icon: IconCubeOutline18,
    title: "Choose which version to call",
    text: "By default, production follows the target app’s live deployment. Git previews use the same branch. You can choose an environment or pin a specific deployment instead.",
    note: "No matching preview? Deploy the target app on that branch or choose another target.",
  },
  {
    label: "Deploy",
    Icon: IconHammer2Outline18,
    title: "Redeploy the calling app",
    text: "Each deployment saves its connection names and target rules. Add, edit, or remove a connection, then redeploy the calling app to apply the change.",
    note: "Existing deployments keep their saved connections. Automatic and environment targets still follow the target app’s deployments.",
  },
  {
    label: "Call",
    Icon: IconTerminalOutline18,
    title: "Use the host variable",
    text: "A connection named api adds API_HOST=api.unkey.internal to the calling app. Read the variable in your code and add the target’s protocol and port. DNS resolves it to ready replicas of the selected deployment.",
  },
];

export function HowConnectionsWorkPanel({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  return (
    <SlidePanel isOpen={isOpen} onClose={onClose} widthClassName="w-120">
      <SlidePanelHeader>
        <div className="flex flex-col gap-0.5">
          <SlidePanelTitle>How connections work</SlidePanelTitle>
          <SlidePanelDescription>
            Connections let your apps call each other over Unkey’s private network.
          </SlidePanelDescription>
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>
      <SlidePanelContent className="overflow-y-auto px-6 pt-4 pb-6">
        {steps.map((step, index) => (
          <Fragment key={step.label}>
            {index > 0 && (
              <div aria-hidden className="ml-6 h-6 border-l border-dashed border-gray-7" />
            )}
            <section className="overflow-hidden rounded-lg border bg-raised">
              <div className="flex border-b border-grayA-4 bg-grayA-2 px-3 py-1 text-2xs font-medium text-gray-11">
                {step.label}
              </div>
              <div className="flex gap-3 px-3 py-3">
                <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised text-gray-11">
                  <step.Icon className="size-3.5" />
                </span>
                <div className="flex min-w-0 flex-col gap-0.5">
                  <h3 className="text-sm font-medium text-gray-12">{step.title}</h3>
                  <p className="text-xs leading-5 text-gray-11">{step.text}</p>
                </div>
              </div>
              {step.note && (
                <p className="border-t border-grayA-4 px-3 py-2 text-xs leading-5 text-gray-11">
                  {step.note}
                </p>
              )}
            </section>
          </Fragment>
        ))}
      </SlidePanelContent>
    </SlidePanel>
  );
}
