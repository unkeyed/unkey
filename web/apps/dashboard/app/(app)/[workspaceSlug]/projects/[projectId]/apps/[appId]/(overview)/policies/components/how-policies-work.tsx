"use client";

import { Logomark } from "@/components/logomark";
import { IconBarsFilterOutline18, IconCubeOutline18 } from "@unkey/icons";
import {
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import type { ReactNode } from "react";
import { POLICY_KINDS, POLICY_TYPES } from "../policy-kinds";
import { POLICY_DOCS_URL, type Rejection } from "../policy-kinds/types";
import { Rejections } from "./rejections";

export function HowPoliciesWorkPanel({
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
          <SlidePanelTitle>How policies work</SlidePanelTitle>
          <SlidePanelDescription>
            Unkey checks each request to your app against your policies.{" "}
            <a
              href={POLICY_DOCS_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="text-gray-12 underline decoration-dotted underline-offset-3"
            >
              Read the docs
            </a>
            .
          </SlidePanelDescription>
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>
      <SlidePanelContent className="overflow-y-auto px-6 pt-4 pb-6">
        <Stage label="Request">
          <Step
            icon={<Logomark className="size-4" />}
            title="Unkey edge"
            text="Requests to your app arrive here first."
          />
        </Stage>
        <StageLink />
        <Stage label="Match">
          <Step
            icon={<IconBarsFilterOutline18 />}
            title="Conditions"
            text="A policy runs only if the request matches all its conditions: path, method, header, query parameter or client IP. With no conditions, it runs on every request. A policy that is off never runs."
          />
        </Stage>
        <StageLink />
        <Stage
          label="Policies"
          note="A rejected request stops here. The policies below it do not run."
        >
          {POLICY_TYPES.map((type) => {
            const kind = POLICY_KINDS[type];
            return (
              <Step
                key={type}
                icon={<kind.Icon />}
                title={kind.label}
                text={kind.does}
                rejects={kind.rejects}
              />
            );
          })}
        </Stage>
        <StageLink />
        <Stage label="Route" note="Unkey removes X-Unkey-* headers that the client sends.">
          <Step
            icon={<IconCubeOutline18 />}
            title="Your app"
            text="Requests that pass every policy go to your app. After Key Auth, the X-Unkey-Principal header holds the verified identity."
          />
        </Stage>
      </SlidePanelContent>
    </SlidePanel>
  );
}

function StageLink() {
  return (
    <div className="flex w-12 justify-center">
      <svg
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        className="pointer-events-none h-6 w-2"
        aria-hidden="true"
      >
        <path
          d="M50,0 V100"
          fill="none"
          strokeWidth={1}
          strokeDasharray="3 3"
          vectorEffect="non-scaling-stroke"
          className="animate-dash-flow stroke-gray-7 motion-reduce:animate-none"
        />
      </svg>
    </div>
  );
}

function Stage({ label, note, children }: { label: string; note?: string; children: ReactNode }) {
  return (
    <section className="overflow-hidden rounded-lg border bg-raised">
      <div className="flex border-b border-grayA-4 bg-grayA-2 px-3 py-1 text-2xs font-medium text-gray-11">
        {label}
      </div>
      <div className="flex flex-col divide-y divide-grayA-4">{children}</div>
      {note ? (
        <p className="border-t border-grayA-4 px-3 py-2 text-xs leading-5 text-gray-11">{note}</p>
      ) : null}
    </section>
  );
}

function Step({
  icon,
  title,
  text,
  rejects,
}: {
  icon: ReactNode;
  title: string;
  text: string;
  rejects?: readonly Rejection[];
}) {
  return (
    <div className="flex gap-3 px-3 py-3">
      <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised text-gray-11 [&_svg]:size-3.5">
        {icon}
      </span>
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="text-sm font-medium text-gray-12">{title}</span>
        <span className="text-xs leading-5 text-gray-11">{text}</span>
        {rejects && rejects.length > 0 ? (
          <span className="pt-1">
            <Rejections rejects={rejects} />
          </span>
        ) : null}
      </span>
    </div>
  );
}
