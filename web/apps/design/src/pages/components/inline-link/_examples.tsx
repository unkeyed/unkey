import { ArrowRight, ArrowUpRight, BookBookmark } from "@unkey/icons";
import { InlineLink } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function IconRightExample() {
  return (
    <Preview>
      <InlineLink label="Open dashboard" href="/dashboard" icon={<ArrowRight />} />
    </Preview>
  );
}

export function IconLeftExample() {
  return (
    <Preview>
      <InlineLink
        label="Read the guide"
        href="/docs/guides/getting-started"
        icon={<BookBookmark />}
        iconPosition="left"
      />
    </Preview>
  );
}

export function ExternalExample() {
  return (
    <Preview>
      <InlineLink
        label="acme.com"
        href="https://acme.com"
        icon={<ArrowUpRight />}
        target="_blank"
        rel="noopener noreferrer"
      />
    </Preview>
  );
}
