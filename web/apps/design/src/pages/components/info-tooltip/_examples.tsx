import { InfoTooltip } from "@unkey/ui";
import { CircleInfo } from "@unkey/icons";
import { Preview } from "../../../components/Preview";

function InfoGlyph() {
  return (
    <span className="inline-flex items-center text-gray-9 hover:text-gray-11">
      <CircleInfo iconSize="md-medium" aria-hidden="true" />
      <span className="sr-only">More info</span>
    </span>
  );
}

export function BasicExample() {
  return (
    <Preview>
      <label
        htmlFor="rate-limit-input"
        className="text-gray-11 text-[13px] flex items-center gap-1.5"
      >
        Requests per minute
        <InfoTooltip
          content="Applies to every key in this namespace. Per-key overrides win."
          asChild
        >
          <InfoGlyph />
        </InfoTooltip>
      </label>
    </Preview>
  );
}

export function VariantsExample() {
  return (
    <Preview>
      <InfoTooltip content="Primary — the default chrome." asChild>
        <InfoGlyph />
      </InfoTooltip>
      <InfoTooltip variant="inverted" content="Inverted — high-contrast on busy surfaces." asChild>
        <InfoGlyph />
      </InfoTooltip>
      <InfoTooltip variant="secondary" content="Secondary — stronger outline, larger text." asChild>
        <InfoGlyph />
      </InfoTooltip>
      <InfoTooltip variant="muted" content="Muted — subtle outline, larger text." asChild>
        <InfoGlyph />
      </InfoTooltip>
    </Preview>
  );
}

export function PositionExample() {
  return (
    <Preview>
      <InfoTooltip
        content="Anchored above the trigger."
        position={{ side: "top" }}
        asChild
      >
        <InfoGlyph />
      </InfoTooltip>
      <InfoTooltip
        content="Anchored below, aligned to the end."
        position={{ side: "bottom", align: "end", sideOffset: 8 }}
        asChild
      >
        <InfoGlyph />
      </InfoTooltip>
      <InfoTooltip
        content="Anchored to the left."
        position={{ side: "left" }}
        asChild
      >
        <InfoGlyph />
      </InfoTooltip>
    </Preview>
  );
}

export function DisabledExample() {
  return (
    <Preview>
      <InfoTooltip content="You will never see this." disabled asChild>
        <InfoGlyph />
      </InfoTooltip>
    </Preview>
  );
}

export function DelayExample() {
  return (
    <Preview>
      <InfoTooltip content="Opens immediately on hover." delayDuration={0} asChild>
        <InfoGlyph />
      </InfoTooltip>
      <InfoTooltip content="Waits 700ms before opening." delayDuration={700} asChild>
        <InfoGlyph />
      </InfoTooltip>
    </Preview>
  );
}
