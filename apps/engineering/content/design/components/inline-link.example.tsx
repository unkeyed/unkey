import { RenderComponentWithSnippet } from "@/app/components/render";
import { ExternalLink } from "@unkey/icons";
import { InlineLink } from "@unkey/ui";

export const InlineLinkBasic = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<p>
  This is a basic <InlineLink href="https://example.com" label="inline link" /> in a
  paragraph.
</p>`}
    >
      <p>
        This is a basic <InlineLink href="https://example.com" label="inline link" /> in a
        paragraph.
      </p>
    </RenderComponentWithSnippet>
  );
};

export const InlineLinkWithIcon = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<p>
  This is an inline link with an icon on the{" "}
  <InlineLink href="https://unkey.com" label="right" icon={<ExternalLink iconSize="md-thin" />} /> and
  on the{" "}
  <InlineLink
    href="https://unkey.com"
    label="left"
    icon={<ExternalLink iconSize="md-thin" />}
    iconPosition="left"
  />
  .
</p>`}
    >
      <p>
        This is an inline link with an icon on the{" "}
        <InlineLink
          href="https://unkey.com"
          label="right"
          icon={<ExternalLink iconSize="md-thin" />}
        />{" "}
        and on the{" "}
        <InlineLink
          href="https://unkey.com"
          label="left"
          icon={<ExternalLink iconSize="md-thin" />}
          iconPosition="left"
        />
        .
      </p>
    </RenderComponentWithSnippet>
  );
};

export const InlineLinkWithTarget = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<p>
  This is an inline link that opens in a{" "}
  <InlineLink
    href="https://unkey.com"
    label="new tab"
    icon={<ExternalLink iconSize="md-thin" />}
    target="_blank"
  />
  .
</p>`}
    >
      <p>
        This is an inline link that opens in a{" "}
        <InlineLink
          href="https://unkey.com"
          label="new tab"
          icon={<ExternalLink iconSize="md-thin" />}
          target="_blank"
        />
        .
      </p>
    </RenderComponentWithSnippet>
  );
};

export const InlineLinkWithCustomClass = () => {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<p>
  This is an inline link with{" "}
  <InlineLink
    href="https://unkey.com"
    label="custom styling"
    className="text-warning-11 hover:text-warning-12"
  />
  .
</p>`}
    >
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
