import { RenderComponentWithSnippet } from "@/app/components/render";
import { InlineLink } from "@unkey/ui";
import { ExternalLink } from "lucide-react";

export const InlineLinkBasic = () => {
  return (
    <RenderComponentWithSnippet>
      <p>
        This is a basic <InlineLink href="https://example.com" label="inline link" /> in a
        paragraph.
      </p>
    </RenderComponentWithSnippet>
  );
};

export const InlineLinkWithIcon = () => {
  return (
    <RenderComponentWithSnippet>
      <p>
        This is an inline link with an icon on the{" "}
        <InlineLink href="https://unkey.com" label="right" icon={<ExternalLink size={14} />} /> and
        on the{" "}
        <InlineLink
          href="https://unkey.com"
          label="left"
          icon={<ExternalLink size={14} />}
          iconPosition="left"
        />
        .
      </p>
    </RenderComponentWithSnippet>
  );
};

export const InlineLinkWithTarget = () => {
  return (
    <RenderComponentWithSnippet>
      <p>
        This is an inline link that opens in a{" "}
        <InlineLink
          href="https://unkey.com"
          label="new tab"
          icon={<ExternalLink size={14} />}
          target
        />
        .
      </p>
    </RenderComponentWithSnippet>
  );
};

export const InlineLinkWithCustomClass = () => {
  return (
    <RenderComponentWithSnippet>
      <p>
        This is an inline link with{" "}
        <InlineLink
          href="https://unkey.com"
          label="custom styling"
          className="text-warning-11 hover:text-warning-12"
        />
        .
      </p>
    </RenderComponentWithSnippet>
  );
};
