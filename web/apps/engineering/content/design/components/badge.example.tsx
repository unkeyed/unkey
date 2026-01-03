import { RenderComponentWithSnippet } from "@/app/components/render";
import { Badge } from "@unkey/ui";

export function DefaultVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-4 justify-center items-center">
    <Badge variant="primary">Primary</Badge>
    <Badge variant="secondary">Secondary</Badge>
    <Badge variant="success">Success</Badge>
    <Badge variant="warning">Warning</Badge>
    <Badge variant="blocked">Blocked</Badge>
    <Badge variant="error">Error</Badge>
</div>`}
    >
      <div className="flex flex-wrap gap-4 justify-center items-center">
        <Badge variant="primary">Primary</Badge>
        <Badge variant="secondary">Secondary</Badge>
        <Badge variant="success">Success</Badge>
        <Badge variant="warning">Warning</Badge>
        <Badge variant="blocked">Blocked</Badge>
        <Badge variant="error">Error</Badge>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function SizeVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-4 justify-center items-center">
    <div className="flex flex-col gap-2 items-center">
        <Badge variant="primary">Default Size</Badge>
        <span className="text-xs text-gray-500">Default</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <Badge variant="primary" size="sm">
            Small Size
        </Badge>
        <span className="text-xs text-gray-500">Small</span>
    </div>
</div>`}
    >
      <div className="flex flex-wrap gap-4 justify-center items-center">
        <div className="flex flex-col gap-2 items-center">
          <Badge variant="primary">Default Size</Badge>
          <span className="text-xs text-gray-500">Default</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <Badge variant="primary" size="sm">
            Small Size
          </Badge>
          <span className="text-xs text-gray-500">Small</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function MonoFont() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-4 justify-center items-center">
    <Badge variant="secondary" font="mono">
        uk_1234567890abcdef
    </Badge>
    <Badge variant="primary" font="mono">
        v1.2.3
    </Badge>
    <Badge variant="success" font="mono">
        200 OK
    </Badge>
    <Badge variant="error" font="mono">
        404 NOT_FOUND
    </Badge>
</div>`}
    >
      <div className="flex flex-wrap gap-4 justify-center items-center">
        <Badge variant="secondary" font="mono">
          uk_1234567890abcdef
        </Badge>
        <Badge variant="primary" font="mono">
          v1.2.3
        </Badge>
        <Badge variant="success" font="mono">
          200 OK
        </Badge>
        <Badge variant="error" font="mono">
          404 NOT_FOUND
        </Badge>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function StatusBadges() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
    <div className="flex items-center gap-3">
        <span className="w-20 text-sm">Active:</span>
        <Badge variant="success">Online</Badge>
    </div>
    <div className="flex items-center gap-3">
        <span className="w-20 text-sm">Warning:</span>
        <Badge variant="warning">Rate Limited</Badge>
    </div>
    <div className="flex items-center gap-3">
        <span className="w-20 text-sm">Blocked:</span>
        <Badge variant="blocked">Suspended</Badge>
    </div>
    <div className="flex items-center gap-3">
        <span className="w-20 text-sm">Error:</span>
        <Badge variant="error">Failed</Badge>
    </div>
    <div className="flex items-center gap-3">
        <span className="w-20 text-sm">Info:</span>
        <Badge variant="secondary">Pending</Badge>
    </div>
</div>`}
    >
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <span className="w-20 text-sm">Active:</span>
          <Badge variant="success">Online</Badge>
        </div>
        <div className="flex items-center gap-3">
          <span className="w-20 text-sm">Warning:</span>
          <Badge variant="warning">Rate Limited</Badge>
        </div>
        <div className="flex items-center gap-3">
          <span className="w-20 text-sm">Blocked:</span>
          <Badge variant="blocked">Suspended</Badge>
        </div>
        <div className="flex items-center gap-3">
          <span className="w-20 text-sm">Error:</span>
          <Badge variant="error">Failed</Badge>
        </div>
        <div className="flex items-center gap-3">
          <span className="w-20 text-sm">Info:</span>
          <Badge variant="secondary">Pending</Badge>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function InteractiveBadges() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-4 justify-center items-center">
    <Badge variant="primary" className="cursor-pointer hover:bg-grayA-5 transition-colors">
        Clickable Tag
    </Badge>
    <Badge
        variant="secondary"
        size="sm"
        className="cursor-pointer hover:bg-grayA-4 transition-colors"
    >
        Category
    </Badge>
    <Badge variant="success" className="cursor-pointer hover:bg-grayA-5 transition-colors">
        ✓ Verified
    </Badge>
    <Badge
        variant="warning"
        font="mono"
        size="sm"
        className="cursor-pointer hover:bg-warningA-5 transition-colors"
    >
        v2.1.0-beta
    </Badge>
</div>`}
    >
      <div className="flex flex-wrap gap-4 justify-center items-center">
        <Badge variant="primary" className="cursor-pointer hover:bg-grayA-5 transition-colors">
          Clickable Tag
        </Badge>
        <Badge
          variant="secondary"
          size="sm"
          className="cursor-pointer hover:bg-grayA-4 transition-colors"
        >
          Category
        </Badge>
        <Badge variant="success" className="cursor-pointer hover:bg-grayA-5 transition-colors">
          ✓ Verified
        </Badge>
        <Badge
          variant="warning"
          font="mono"
          size="sm"
          className="cursor-pointer hover:bg-warningA-5 transition-colors"
        >
          v2.1.0-beta
        </Badge>
      </div>
    </RenderComponentWithSnippet>
  );
}
