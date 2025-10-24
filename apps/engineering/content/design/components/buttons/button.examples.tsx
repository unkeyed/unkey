import { RenderComponentWithSnippet } from "@/app/components/render";
import { Magnifier, Plus, Trash } from "@unkey/icons";
import { Button } from "@unkey/ui";

// Basic Variants

export const PrimaryExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={` <div className="flex flex-col gap-6">
    <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
            <Button>Default</Button>
            <Button className="!bg-accent-12/90">Hover</Button>
            <Button className="!ring-4 !ring-gray-7">Focus</Button>
            <Button loading loadingLabel="Loading...">Loading</Button>
            <Button disabled>Disabled</Button>
        </div>
    </div>
    <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Button>Default</Button>
            <Button className="!bg-accent-12/90">Hover</Button>
            <Button className="!ring-4 !ring-gray-7">Focus</Button>
            <Button loading>Loading</Button>
            <Button disabled>Disabled</Button>
        </div>
    </div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button>Default</Button>
          <Button className="!bg-accent-12/90">Hover</Button>
          <Button className="!ring-4 !ring-gray-7">Focus</Button>
          <Button loading loadingLabel="Loading...">
            Loading
          </Button>
          <Button disabled>Disabled</Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button>Default</Button>
          <Button className="!bg-accent-12/90">Hover</Button>
          <Button className="!ring-4 !ring-gray-7">Focus</Button>
          <Button loading>Loading</Button>
          <Button disabled>Disabled</Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const OutlineExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="outline">Default</Button>
        <Button variant="outline" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            className="!ring-4 !ring-gray-7 !border-grayA-12"
        >
            Focus
        </Button>
        <Button variant="outline" loading>
            Loading
        </Button>
        <Button variant="outline" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="outline">Default</Button>
        <Button variant="outline" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            className="!ring-4 !ring-gray-7 !border-grayA-12"
        >
            Focus
        </Button>
        <Button variant="outline" loading>
            Loading
        </Button>
        <Button variant="outline" disabled>
            Disabled
        </Button>
    </div>
    </div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="outline">Default</Button>
          <Button variant="outline" className="!bg-grayA-2">
            Hover
          </Button>
          <Button variant="outline" className="!ring-4 !ring-gray-7 !border-grayA-12">
            Focus
          </Button>
          <Button variant="outline" loading>
            Loading
          </Button>
          <Button variant="outline" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="outline">Default</Button>
          <Button variant="outline" className="!bg-grayA-2">
            Hover
          </Button>
          <Button variant="outline" className="!ring-4 !ring-gray-7 !border-grayA-12">
            Focus
          </Button>
          <Button variant="outline" loading>
            Loading
          </Button>
          <Button variant="outline" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const GhostExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
        <div>
            <h4 className="text-sm font-medium mb-2">Light</h4>
            <div className="flex flex-wrap items-center gap-4">
                <Button variant="ghost">Default</Button>
                <Button variant="ghost" className="!bg-grayA-4">
                    Hover
                </Button>
                <Button
                    variant="ghost"
                    className="!ring-4 !ring-gray-7 !border-grayA-12"
                >
                    Focus
                </Button>
                <Button variant="ghost" loading>
                    Loading
                </Button>
                <Button variant="ghost" disabled>
                    Disabled
                </Button>
            </div>
        </div>
        <div>
            <h4 className="text-sm font-medium mb-2">Dark</h4>
            <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
                <Button variant="ghost">Default</Button>
                <Button variant="ghost" className="!bg-grayA-4">
                    Hover
                </Button>
                <Button
                    variant="ghost"
                    className="!ring-4 !ring-gray-7 !border-grayA-12"
                >
                    Focus
                </Button>
                <Button variant="ghost" loading>
                    Loading
                </Button>
                <Button variant="ghost" disabled>
                    Disabled
                </Button>
            </div>
        </div>
    </div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="ghost">Default</Button>
          <Button variant="ghost" className="!bg-grayA-4">
            Hover
          </Button>
          <Button variant="ghost" className="!ring-4 !ring-gray-7 !border-grayA-12">
            Focus
          </Button>
          <Button variant="ghost" loading>
            Loading
          </Button>
          <Button variant="ghost" disabled>
            Disabled
          </Button>
        </div>
      </div>

      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="ghost">Default</Button>
          <Button variant="ghost" className="!bg-grayA-4">
            Hover
          </Button>
          <Button variant="ghost" className="!ring-4 !ring-gray-7 !border-grayA-12">
            Focus
          </Button>
          <Button variant="ghost" loading>
            Loading
          </Button>
          <Button variant="ghost" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

// Color Variants

export const DangerPrimaryExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="primary" color="danger">
            Default
        </Button>
        <Button variant="primary" color="danger" className="!bg-error-10">
            Hover
        </Button>
        <Button
            variant="primary"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="danger" loading>
            Loading
        </Button>
        <Button variant="primary" color="danger" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="primary" color="danger">
            Default
        </Button>
        <Button variant="primary" color="danger" className="!bg-error-10">
            Hover
        </Button>
        <Button
            variant="primary"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="danger" loading>
            Loading
        </Button>
        <Button variant="primary" color="danger" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="primary" color="danger">
            Default
          </Button>
          <Button variant="primary" color="danger" className="!bg-error-10">
            Hover
          </Button>
          <Button
            variant="primary"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
          >
            Focus
          </Button>
          <Button variant="primary" color="danger" loading>
            Loading
          </Button>
          <Button variant="primary" color="danger" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="primary" color="danger">
            Default
          </Button>
          <Button variant="primary" color="danger" className="!bg-error-10">
            Hover
          </Button>
          <Button
            variant="primary"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
          >
            Focus
          </Button>
          <Button variant="primary" color="danger" loading>
            Loading
          </Button>
          <Button variant="primary" color="danger" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const DangerOutlineExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="outline" color="danger">
            Default
        </Button>
        <Button variant="outline" color="danger" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="danger" loading>
            Loading
        </Button>
        <Button variant="outline" color="danger" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="outline" color="danger">
            Default
        </Button>
        <Button variant="outline" color="danger" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="danger" loading>
            Loading
        </Button>
        <Button variant="outline" color="danger" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="outline" color="danger">
            Default
          </Button>
          <Button variant="outline" color="danger" className="!bg-grayA-2">
            Hover
          </Button>
          <Button
            variant="outline"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
          >
            Focus
          </Button>
          <Button variant="outline" color="danger" loading>
            Loading
          </Button>
          <Button variant="outline" color="danger" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="outline" color="danger">
            Default
          </Button>
          <Button variant="outline" color="danger" className="!bg-grayA-2">
            Hover
          </Button>
          <Button
            variant="outline"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
          >
            Focus
          </Button>
          <Button variant="outline" color="danger" loading>
            Loading
          </Button>
          <Button variant="outline" color="danger" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const DangerGhostExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="ghost" color="danger">
            Default
        </Button>
        <Button variant="ghost" color="danger" className="!bg-error-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="danger" loading>
            Loading
        </Button>
        <Button variant="ghost" color="danger" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="ghost" color="danger">
            Default
        </Button>
        <Button variant="ghost" color="danger" className="!bg-error-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="danger"
            className="!ring-4 !ring-error-7 !border-error-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="danger" loading>
            Loading
        </Button>
        <Button variant="ghost" color="danger" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="ghost" color="danger">
            Default
          </Button>
          <Button variant="ghost" color="danger" className="!bg-error-3">
            Hover
          </Button>
          <Button variant="ghost" color="danger" className="!ring-4 !ring-error-7 !border-error-11">
            Focus
          </Button>
          <Button variant="ghost" color="danger" loading>
            Loading
          </Button>
          <Button variant="ghost" color="danger" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="ghost" color="danger">
            Default
          </Button>
          <Button variant="ghost" color="danger" className="!bg-error-3">
            Hover
          </Button>
          <Button variant="ghost" color="danger" className="!ring-4 !ring-error-7 !border-error-11">
            Focus
          </Button>
          <Button variant="ghost" color="danger" loading>
            Loading
          </Button>
          <Button variant="ghost" color="danger" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const WarningPrimaryExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="primary" color="warning">
            Default
        </Button>
        <Button variant="primary" color="warning" className="!bg-warning-8/90">
            Hover
        </Button>
        <Button
            variant="primary"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="warning" loading>
            Loading
        </Button>
        <Button variant="primary" color="warning" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="primary" color="warning">
            Default
        </Button>
        <Button variant="primary" color="warning" className="!bg-warning-8/90">
            Hover
        </Button>
        <Button
            variant="primary"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="warning" loading>
            Loading
        </Button>
        <Button variant="primary" color="warning" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="primary" color="warning">
            Default
          </Button>
          <Button variant="primary" color="warning" className="!bg-warning-8/90">
            Hover
          </Button>
          <Button
            variant="primary"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
          >
            Focus
          </Button>
          <Button variant="primary" color="warning" loading>
            Loading
          </Button>
          <Button variant="primary" color="warning" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="primary" color="warning">
            Default
          </Button>
          <Button variant="primary" color="warning" className="!bg-warning-8/90">
            Hover
          </Button>
          <Button
            variant="primary"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
          >
            Focus
          </Button>
          <Button variant="primary" color="warning" loading>
            Loading
          </Button>
          <Button variant="primary" color="warning" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const WarningOutlineExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={` <div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="outline" color="warning">
            Default
        </Button>
        <Button variant="outline" color="warning" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="warning" loading>
            Loading
        </Button>
        <Button variant="outline" color="warning" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="outline" color="warning">
            Default
        </Button>
        <Button variant="outline" color="warning" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="warning" loading>
            Loading
        </Button>
        <Button variant="outline" color="warning" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="outline" color="warning">
            Default
          </Button>
          <Button variant="outline" color="warning" className="!bg-grayA-2">
            Hover
          </Button>
          <Button
            variant="outline"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
          >
            Focus
          </Button>
          <Button variant="outline" color="warning" loading>
            Loading
          </Button>
          <Button variant="outline" color="warning" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="outline" color="warning">
            Default
          </Button>
          <Button variant="outline" color="warning" className="!bg-grayA-2">
            Hover
          </Button>
          <Button
            variant="outline"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
          >
            Focus
          </Button>
          <Button variant="outline" color="warning" loading>
            Loading
          </Button>
          <Button variant="outline" color="warning" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const WarningGhostExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="ghost" color="warning">
            Default
        </Button>
        <Button variant="ghost" color="warning" className="!bg-warning-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="warning" loading>
            Loading
        </Button>
        <Button variant="ghost" color="warning" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="ghost" color="warning">
            Default
        </Button>
        <Button variant="ghost" color="warning" className="!bg-warning-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="warning" loading>
            Loading
        </Button>
        <Button variant="ghost" color="warning" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="ghost" color="warning">
            Default
          </Button>
          <Button variant="ghost" color="warning" className="!bg-warning-3">
            Hover
          </Button>
          <Button
            variant="ghost"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
          >
            Focus
          </Button>
          <Button variant="ghost" color="warning" loading>
            Loading
          </Button>
          <Button variant="ghost" color="warning" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="ghost" color="warning">
            Default
          </Button>
          <Button variant="ghost" color="warning" className="!bg-warning-3">
            Hover
          </Button>
          <Button
            variant="ghost"
            color="warning"
            className="!ring-4 !ring-warning-6 !border-warning-11"
          >
            Focus
          </Button>
          <Button variant="ghost" color="warning" loading>
            Loading
          </Button>
          <Button variant="ghost" color="warning" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const SuccessPrimaryExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="primary" color="success">
            Default
        </Button>
        <Button variant="primary" color="success" className="!bg-success-9/90">
            Hover
        </Button>
        <Button
            variant="primary"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="success" loading>
            Loading
        </Button>
        <Button variant="primary" color="success" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="primary" color="success">
            Default
        </Button>
        <Button variant="primary" color="success" className="!bg-success-9/90">
            Hover
        </Button>
        <Button
            variant="primary"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="success" loading>
            Loading
        </Button>
        <Button variant="primary" color="success" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="primary" color="success">
            Default
          </Button>
          <Button variant="primary" color="success" className="!bg-success-9/90">
            Hover
          </Button>
          <Button
            variant="primary"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
          >
            Focus
          </Button>
          <Button variant="primary" color="success" loading>
            Loading
          </Button>
          <Button variant="primary" color="success" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="primary" color="success">
            Default
          </Button>
          <Button variant="primary" color="success" className="!bg-success-9/90">
            Hover
          </Button>
          <Button
            variant="primary"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
          >
            Focus
          </Button>
          <Button variant="primary" color="success" loading>
            Loading
          </Button>
          <Button variant="primary" color="success" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const SuccessOutlineExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="outline" color="success">
            Default
        </Button>
        <Button variant="outline" color="success" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="success" loading>
            Loading
        </Button>
        <Button variant="outline" color="success" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="outline" color="success">
            Default
        </Button>
        <Button variant="outline" color="success" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="success" loading>
            Loading
        </Button>
        <Button variant="outline" color="success" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="outline" color="success">
            Default
          </Button>
          <Button variant="outline" color="success" className="!bg-grayA-2">
            Hover
          </Button>
          <Button
            variant="outline"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
          >
            Focus
          </Button>
          <Button variant="outline" color="success" loading>
            Loading
          </Button>
          <Button variant="outline" color="success" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="outline" color="success">
            Default
          </Button>
          <Button variant="outline" color="success" className="!bg-grayA-2">
            Hover
          </Button>
          <Button
            variant="outline"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
          >
            Focus
          </Button>
          <Button variant="outline" color="success" loading>
            Loading
          </Button>
          <Button variant="outline" color="success" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const SuccessGhostExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="ghost" color="success">
            Default
        </Button>
        <Button variant="ghost" color="success" className="!bg-success-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="success" loading>
            Loading
        </Button>
        <Button variant="ghost" color="success" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="ghost" color="success">
            Default
        </Button>
        <Button variant="ghost" color="success" className="!bg-success-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="success" loading>
            Loading
        </Button>
        <Button variant="ghost" color="success" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="ghost" color="success">
            Default
          </Button>
          <Button variant="ghost" color="success" className="!bg-success-3">
            Hover
          </Button>
          <Button
            variant="ghost"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
          >
            Focus
          </Button>
          <Button variant="ghost" color="success" loading>
            Loading
          </Button>
          <Button variant="ghost" color="success" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="ghost" color="success">
            Default
          </Button>
          <Button variant="ghost" color="success" className="!bg-success-3">
            Hover
          </Button>
          <Button
            variant="ghost"
            color="success"
            className="!ring-4 !ring-success-6 !border-success-11"
          >
            Focus
          </Button>
          <Button variant="ghost" color="success" loading>
            Loading
          </Button>
          <Button variant="ghost" color="success" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const InfoPrimaryExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="primary" color="info">
            Default
        </Button>
        <Button variant="primary" color="info" className="!bg-info-9/90">
            Hover
        </Button>
        <Button
            variant="primary"
            color="info"
            className="!ring-4 !ring-info-6 !border-info-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="info" loading>
            Loading
        </Button>
        <Button variant="primary" color="info" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="primary" color="info">
            Default
        </Button>
        <Button variant="primary" color="info" className="!bg-info-9/90">
            Hover
        </Button>
        <Button
            variant="primary"
            color="info"
            className="!ring-4 !ring-info-6 !border-info-11"
        >
            Focus
        </Button>
        <Button variant="primary" color="info" loading>
            Loading
        </Button>
        <Button variant="primary" color="info" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="primary" color="info">
            Default
          </Button>
          <Button variant="primary" color="info" className="!bg-info-9/90">
            Hover
          </Button>
          <Button variant="primary" color="info" className="!ring-4 !ring-info-6 !border-info-11">
            Focus
          </Button>
          <Button variant="primary" color="info" loading>
            Loading
          </Button>
          <Button variant="primary" color="info" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="primary" color="info">
            Default
          </Button>
          <Button variant="primary" color="info" className="!bg-info-9/90">
            Hover
          </Button>
          <Button variant="primary" color="info" className="!ring-4 !ring-info-6 !border-info-11">
            Focus
          </Button>
          <Button variant="primary" color="info" loading>
            Loading
          </Button>
          <Button variant="primary" color="info" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const InfoOutlineExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="outline" color="info">
            Default
        </Button>
        <Button variant="outline" color="info" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="info"
            className="!ring-4 !ring-info-6 !border-info-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="info" loading>
            Loading
        </Button>
        <Button variant="outline" color="info" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="outline" color="info">
            Default
        </Button>
        <Button variant="outline" color="info" className="!bg-grayA-2">
            Hover
        </Button>
        <Button
            variant="outline"
            color="info"
            className="!ring-4 !ring-info-6 !border-info-11"
        >
            Focus
        </Button>
        <Button variant="outline" color="info" loading>
            Loading
        </Button>
        <Button variant="outline" color="info" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="outline" color="info">
            Default
          </Button>
          <Button variant="outline" color="info" className="!bg-grayA-2">
            Hover
          </Button>
          <Button variant="outline" color="info" className="!ring-4 !ring-info-6 !border-info-11">
            Focus
          </Button>
          <Button variant="outline" color="info" loading>
            Loading
          </Button>
          <Button variant="outline" color="info" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="outline" color="info">
            Default
          </Button>
          <Button variant="outline" color="info" className="!bg-grayA-2">
            Hover
          </Button>
          <Button variant="outline" color="info" className="!ring-4 !ring-info-6 !border-info-11">
            Focus
          </Button>
          <Button variant="outline" color="info" loading>
            Loading
          </Button>
          <Button variant="outline" color="info" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const InfoGhostExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button variant="ghost" color="info">
            Default
        </Button>
        <Button variant="ghost" color="info" className="!bg-info-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="info"
            className="!ring-4 !ring-info-6 !border-info-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="info" loading>
            Loading
        </Button>
        <Button variant="ghost" color="info" disabled>
            Disabled
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button variant="ghost" color="info">
            Default
        </Button>
        <Button variant="ghost" color="info" className="!bg-info-3">
            Hover
        </Button>
        <Button
            variant="ghost"
            color="info"
            className="!ring-4 !ring-info-6 !border-info-11"
        >
            Focus
        </Button>
        <Button variant="ghost" color="info" loading>
            Loading
        </Button>
        <Button variant="ghost" color="info" disabled>
            Disabled
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button variant="ghost" color="info">
            Default
          </Button>
          <Button variant="ghost" color="info" className="!bg-info-3">
            Hover
          </Button>
          <Button variant="ghost" color="info" className="!ring-4 !ring-info-6 !border-info-11">
            Focus
          </Button>
          <Button variant="ghost" color="info" loading>
            Loading
          </Button>
          <Button variant="ghost" color="info" disabled>
            Disabled
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button variant="ghost" color="info">
            Default
          </Button>
          <Button variant="ghost" color="info" className="!bg-info-3">
            Hover
          </Button>
          <Button variant="ghost" color="info" className="!ring-4 !ring-info-6 !border-info-11">
            Focus
          </Button>
          <Button variant="ghost" color="info" loading>
            Loading
          </Button>
          <Button variant="ghost" color="info" disabled>
            Disabled
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

// Size Variants

export const SizeVariantsExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button size="sm">Small</Button>
        <Button size="md">Medium</Button>
        <Button size="lg">Large</Button>
        <Button size="xlg">Extra Large</Button>
        <Button size="2xlg">2X Large</Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button size="sm">Small</Button>
        <Button size="md">Medium</Button>
        <Button size="lg">Large</Button>
        <Button size="xlg">Extra Large</Button>
        <Button size="2xlg">2X Large</Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button size="sm">Small</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
          <Button size="xlg">Extra Large</Button>
          <Button size="2xlg">2X Large</Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button size="sm">Small</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
          <Button size="xlg">Extra Large</Button>
          <Button size="2xlg">2X Large</Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

// Special Features

export const WithIconsExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
        <Button>
            <span>Create</span>
            <Plus />
        </Button>
        <Button variant="outline">
            <Magnifier />
            <span>Search</span>
        </Button>
        <Button variant="ghost" color="danger">
            <Trash />
            <span>Delete</span>
        </Button>
    </div>
</div>
<div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Button>
            <span>Create</span>
            <Plus />
        </Button>
        <Button variant="outline">
            <Magnifier />
            <span>Search</span>
        </Button>
        <Button variant="ghost" color="danger">
            <Trash />
            <span>Delete</span>
        </Button>
    </div>
</div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex flex-wrap items-center gap-4">
          <Button>
            <span>Create</span>
            <Plus />
          </Button>
          <Button variant="outline">
            <Magnifier />
            <span>Search</span>
          </Button>
          <Button variant="ghost" color="danger">
            <Trash />
            <span>Delete</span>
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
          <Button>
            <span>Create</span>
            <Plus />
          </Button>
          <Button variant="outline">
            <Magnifier />
            <span>Search</span>
          </Button>
          <Button variant="ghost" color="danger">
            <Trash />
            <span>Delete</span>
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);

export const ShapeVariantsExample = () => (
  <RenderComponentWithSnippet
    customCodeSnippet={`<div className="flex flex-col gap-6">
    <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex items-center gap-4">
            <Button shape="square">
                <Plus />
            </Button>
            <Button shape="square" variant="outline">
                <Magnifier />
            </Button>
            <Button shape="square" variant="ghost" color="danger">
                <Trash />
            </Button>
        </div>
    </div>
    <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex items-center gap-4 dark">
            <Button shape="square">
                <Plus />
            </Button>
            <Button shape="square" variant="outline">
                <Magnifier />
            </Button>
            <Button shape="square" variant="ghost" color="danger">
                <Trash />
            </Button>
        </div>
    </div>
</div>`}
  >
    <div className="flex flex-col gap-6">
      <div>
        <h4 className="text-sm font-medium mb-2">Light</h4>
        <div className="flex items-center gap-4">
          <Button shape="square">
            <Plus />
          </Button>
          <Button shape="square" variant="outline">
            <Magnifier />
          </Button>
          <Button shape="square" variant="ghost" color="danger">
            <Trash />
          </Button>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Dark</h4>
        <div className="bg-black p-4 rounded-md flex items-center gap-4 dark">
          <Button shape="square">
            <Plus />
          </Button>
          <Button shape="square" variant="outline">
            <Magnifier />
          </Button>
          <Button shape="square" variant="ghost" color="danger">
            <Trash />
          </Button>
        </div>
      </div>
    </div>
  </RenderComponentWithSnippet>
);
