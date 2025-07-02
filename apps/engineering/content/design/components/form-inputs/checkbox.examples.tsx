"use client";

import { RenderComponentWithSnippet } from "@/app/components/render";
import { Checkbox } from "@unkey/ui";

export function CheckboxBasicVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="primary-default" />
      <Checkbox id="primary-checked" checked />
      <Checkbox id="primary-focus" className="!ring-4 !ring-gray-6" />
      <Checkbox id="primary-disabled" disabled />
      <Checkbox id="primary-disabled-checked" disabled checked />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="primary-dark-default" />
      <Checkbox id="primary-dark-checked" checked />
      <Checkbox id="primary-dark-focus" className="!ring-4 !ring-gray-6" />
      <Checkbox id="primary-dark-disabled" disabled />
      <Checkbox id="primary-dark-disabled-checked" disabled checked />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="primary-default" />
            <Checkbox id="primary-checked" checked />
            <Checkbox id="primary-focus" className="!ring-4 !ring-gray-6" />
            <Checkbox id="primary-disabled" disabled />
            <Checkbox id="primary-disabled-checked" disabled checked />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="primary-dark-default" />
            <Checkbox id="primary-dark-checked" checked />
            <Checkbox id="primary-dark-focus" className="!ring-4 !ring-gray-6" />
            <Checkbox id="primary-dark-disabled" disabled />
            <Checkbox id="primary-dark-disabled-checked" disabled checked />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxOutlineVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
  <h4 className="text-sm font-medium mb-2">Light</h4>
  <div className="flex flex-wrap items-center gap-4">
    <Checkbox id="outline-default" variant="outline" />
    <Checkbox id="outline-checked" variant="outline" checked />
    <Checkbox
      id="outline-focus"
      variant="outline"
      className="!ring-4 !ring-gray-6 !border-grayA-12"
    />
    <Checkbox id="outline-disabled" variant="outline" disabled />
    <Checkbox id="outline-disabled-checked" variant="outline" disabled checked />
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="outline-dark-default" variant="outline" />
      <Checkbox id="outline-dark-checked" variant="outline" checked />
      <Checkbox
        id="outline-dark-focus"
        variant="outline"
        className="!ring-4 !ring-gray-6 !border-grayA-12"
      />
      <Checkbox id="outline-dark-disabled" variant="outline" disabled />
      <Checkbox id="outline-dark-disabled-checked" variant="outline" disabled checked />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="outline-default" variant="outline" />
            <Checkbox id="outline-checked" variant="outline" checked />
            <Checkbox
              id="outline-focus"
              variant="outline"
              className="!ring-4 !ring-gray-6 !border-grayA-12"
            />
            <Checkbox id="outline-disabled" variant="outline" disabled />
            <Checkbox id="outline-disabled-checked" variant="outline" disabled checked />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="outline-dark-default" variant="outline" />
            <Checkbox id="outline-dark-checked" variant="outline" checked />
            <Checkbox
              id="outline-dark-focus"
              variant="outline"
              className="!ring-4 !ring-gray-6 !border-grayA-12"
            />
            <Checkbox id="outline-dark-disabled" variant="outline" disabled />
            <Checkbox id="outline-dark-disabled-checked" variant="outline" disabled checked />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxGhostVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="ghost-default" variant="ghost" />
      <Checkbox id="ghost-checked" variant="ghost" checked />
      <Checkbox id="ghost-hover" variant="ghost" className="!bg-grayA-2" />
      <Checkbox
        id="ghost-focus"
        variant="ghost"
        className="!ring-4 !ring-gray-6 !border-grayA-12"
      />
      <Checkbox id="ghost-disabled" variant="ghost" disabled />
      <Checkbox id="ghost-disabled-checked" variant="ghost" disabled checked />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="ghost-dark-default" variant="ghost" />
      <Checkbox id="ghost-dark-checked" variant="ghost" checked />
      <Checkbox id="ghost-dark-hover" variant="ghost" className="!bg-grayA-2" />
      <Checkbox
        id="ghost-dark-focus"
        variant="ghost"
        className="!ring-4 !ring-gray-6 !border-grayA-12"
      />
      <Checkbox id="ghost-dark-disabled" variant="ghost" disabled />
      <Checkbox id="ghost-dark-disabled-checked" variant="ghost" disabled checked />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="ghost-default" variant="ghost" />
            <Checkbox id="ghost-checked" variant="ghost" checked />
            <Checkbox id="ghost-hover" variant="ghost" className="!bg-grayA-2" />
            <Checkbox
              id="ghost-focus"
              variant="ghost"
              className="!ring-4 !ring-gray-6 !border-grayA-12"
            />
            <Checkbox id="ghost-disabled" variant="ghost" disabled />
            <Checkbox id="ghost-disabled-checked" variant="ghost" disabled checked />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="ghost-dark-default" variant="ghost" />
            <Checkbox id="ghost-dark-checked" variant="ghost" checked />
            <Checkbox id="ghost-dark-hover" variant="ghost" className="!bg-grayA-2" />
            <Checkbox
              id="ghost-dark-focus"
              variant="ghost"
              className="!ring-4 !ring-gray-6 !border-grayA-12"
            />
            <Checkbox id="ghost-dark-disabled" variant="ghost" disabled />
            <Checkbox id="ghost-dark-disabled-checked" variant="ghost" disabled checked />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxDangerPrimary() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="danger-primary-default" variant="primary" color="danger" />
      <Checkbox id="danger-primary-checked" variant="primary" color="danger" checked />
      <Checkbox
        id="danger-primary-focus"
        variant="primary"
        color="danger"
        className="!ring-4 !ring-error-6"
      />
      <Checkbox id="danger-primary-disabled" variant="primary" color="danger" disabled />
      <Checkbox
        id="danger-primary-disabled-checked"
        variant="primary"
        color="danger"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="danger-primary-dark-default" variant="primary" color="danger" />
      <Checkbox id="danger-primary-dark-checked" variant="primary" color="danger" checked />
      <Checkbox
        id="danger-primary-dark-focus"
        variant="primary"
        color="danger"
        className="!ring-4 !ring-error-6"
      />
      <Checkbox id="danger-primary-dark-disabled" variant="primary" color="danger" disabled />
      <Checkbox
        id="danger-primary-dark-disabled-checked"
        variant="primary"
        color="danger"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="danger-primary-default" variant="primary" color="danger" />
            <Checkbox id="danger-primary-checked" variant="primary" color="danger" checked />
            <Checkbox
              id="danger-primary-focus"
              variant="primary"
              color="danger"
              className="!ring-4 !ring-error-6"
            />
            <Checkbox id="danger-primary-disabled" variant="primary" color="danger" disabled />
            <Checkbox
              id="danger-primary-disabled-checked"
              variant="primary"
              color="danger"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="danger-primary-dark-default" variant="primary" color="danger" />
            <Checkbox id="danger-primary-dark-checked" variant="primary" color="danger" checked />
            <Checkbox
              id="danger-primary-dark-focus"
              variant="primary"
              color="danger"
              className="!ring-4 !ring-error-6"
            />
            <Checkbox id="danger-primary-dark-disabled" variant="primary" color="danger" disabled />
            <Checkbox
              id="danger-primary-dark-disabled-checked"
              variant="primary"
              color="danger"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxDangerOutline() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="danger-outline-default" variant="outline" color="danger" />
      <Checkbox id="danger-outline-checked" variant="outline" color="danger" checked />
      <Checkbox
        id="danger-outline-focus"
        variant="outline"
        color="danger"
        className="!ring-4 !ring-error-6 !border-error-9"
      />
      <Checkbox id="danger-outline-disabled" variant="outline" color="danger" disabled />
      <Checkbox
        id="danger-outline-disabled-checked"
        variant="outline"
        color="danger"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="danger-outline-dark-default" variant="outline" color="danger" />
      <Checkbox id="danger-outline-dark-checked" variant="outline" color="danger" checked />
      <Checkbox
        id="danger-outline-dark-focus"
        variant="outline"
        color="danger"
        className="!ring-4 !ring-error-6 !border-error-9"
      />
      <Checkbox id="danger-outline-dark-disabled" variant="outline" color="danger" disabled />
      <Checkbox
        id="danger-outline-dark-disabled-checked"
        variant="outline"
        color="danger"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="danger-outline-default" variant="outline" color="danger" />
            <Checkbox id="danger-outline-checked" variant="outline" color="danger" checked />
            <Checkbox
              id="danger-outline-focus"
              variant="outline"
              color="danger"
              className="!ring-4 !ring-error-6 !border-error-9"
            />
            <Checkbox id="danger-outline-disabled" variant="outline" color="danger" disabled />
            <Checkbox
              id="danger-outline-disabled-checked"
              variant="outline"
              color="danger"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="danger-outline-dark-default" variant="outline" color="danger" />
            <Checkbox id="danger-outline-dark-checked" variant="outline" color="danger" checked />
            <Checkbox
              id="danger-outline-dark-focus"
              variant="outline"
              color="danger"
              className="!ring-4 !ring-error-6 !border-error-9"
            />
            <Checkbox id="danger-outline-dark-disabled" variant="outline" color="danger" disabled />
            <Checkbox
              id="danger-outline-dark-disabled-checked"
              variant="outline"
              color="danger"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxDangerGhost() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`
<div className="flex flex-col gap-6">
    <div>
      <h4 className="text-sm font-medium mb-2">Light</h4>
      <div className="flex flex-wrap items-center gap-4">
        <Checkbox id="danger-ghost-default" variant="ghost" color="danger" />
        <Checkbox id="danger-ghost-checked" variant="ghost" color="danger" checked />
        <Checkbox
          id="danger-ghost-hover"
          variant="ghost"
          color="danger"
          className="!bg-error-3"
        />
        <Checkbox
          id="danger-ghost-focus"
          variant="ghost"
          color="danger"
          className="!ring-4 !ring-error-6 !border-error-9"
        />
        <Checkbox id="danger-ghost-disabled" variant="ghost" color="danger" disabled />
        <Checkbox
          id="danger-ghost-disabled-checked"
          variant="ghost"
          color="danger"
          disabled
          checked
        />
      </div>
    </div>

    <div>
      <h4 className="text-sm font-medium mb-2">Dark</h4>
      <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
        <Checkbox id="danger-ghost-dark-default" variant="ghost" color="danger" />
        <Checkbox id="danger-ghost-dark-checked" variant="ghost" color="danger" checked />
        <Checkbox
          id="danger-ghost-dark-hover"
          variant="ghost"
          color="danger"
          className="!bg-error-3"
        />
        <Checkbox
          id="danger-ghost-dark-focus"
          variant="ghost"
          color="danger"
          className="!ring-4 !ring-error-6 !border-error-9"
        />
        <Checkbox id="danger-ghost-dark-disabled" variant="ghost" color="danger" disabled />
        <Checkbox
          id="danger-ghost-dark-disabled-checked"
          variant="ghost"
          color="danger"
          disabled
          checked
        />
      </div>
    </div>
  </div>
    `}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="danger-ghost-default" variant="ghost" color="danger" />
            <Checkbox id="danger-ghost-checked" variant="ghost" color="danger" checked />
            <Checkbox
              id="danger-ghost-hover"
              variant="ghost"
              color="danger"
              className="!bg-error-3"
            />
            <Checkbox
              id="danger-ghost-focus"
              variant="ghost"
              color="danger"
              className="!ring-4 !ring-error-6 !border-error-9"
            />
            <Checkbox id="danger-ghost-disabled" variant="ghost" color="danger" disabled />
            <Checkbox
              id="danger-ghost-disabled-checked"
              variant="ghost"
              color="danger"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="danger-ghost-dark-default" variant="ghost" color="danger" />
            <Checkbox id="danger-ghost-dark-checked" variant="ghost" color="danger" checked />
            <Checkbox
              id="danger-ghost-dark-hover"
              variant="ghost"
              color="danger"
              className="!bg-error-3"
            />
            <Checkbox
              id="danger-ghost-dark-focus"
              variant="ghost"
              color="danger"
              className="!ring-4 !ring-error-6 !border-error-9"
            />
            <Checkbox id="danger-ghost-dark-disabled" variant="ghost" color="danger" disabled />
            <Checkbox
              id="danger-ghost-dark-disabled-checked"
              variant="ghost"
              color="danger"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxWarningPrimary() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="warning-primary-default" variant="primary" color="warning" />
      <Checkbox id="warning-primary-checked" variant="primary" color="warning" checked />
      <Checkbox
        id="warning-primary-focus"
        variant="primary"
        color="warning"
        className="!ring-4 !ring-warning-6"
      />
      <Checkbox id="warning-primary-disabled" variant="primary" color="warning" disabled />
      <Checkbox
        id="warning-primary-disabled-checked"
        variant="primary"
        color="warning"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="warning-primary-dark-default" variant="primary" color="warning" />
      <Checkbox id="warning-primary-dark-checked" variant="primary" color="warning" checked />
      <Checkbox
        id="warning-primary-dark-focus"
        variant="primary"
        color="warning"
        className="!ring-4 !ring-warning-6"
      />
      <Checkbox
        id="warning-primary-dark-disabled"
        variant="primary"
        color="warning"
        disabled
      />
      <Checkbox
        id="warning-primary-dark-disabled-checked"
        variant="primary"
        color="warning"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="warning-primary-default" variant="primary" color="warning" />
            <Checkbox id="warning-primary-checked" variant="primary" color="warning" checked />
            <Checkbox
              id="warning-primary-focus"
              variant="primary"
              color="warning"
              className="!ring-4 !ring-warning-6"
            />
            <Checkbox id="warning-primary-disabled" variant="primary" color="warning" disabled />
            <Checkbox
              id="warning-primary-disabled-checked"
              variant="primary"
              color="warning"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="warning-primary-dark-default" variant="primary" color="warning" />
            <Checkbox id="warning-primary-dark-checked" variant="primary" color="warning" checked />
            <Checkbox
              id="warning-primary-dark-focus"
              variant="primary"
              color="warning"
              className="!ring-4 !ring-warning-6"
            />
            <Checkbox
              id="warning-primary-dark-disabled"
              variant="primary"
              color="warning"
              disabled
            />
            <Checkbox
              id="warning-primary-dark-disabled-checked"
              variant="primary"
              color="warning"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxWarningOutline() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="warning-outline-default" variant="outline" color="warning" />
      <Checkbox id="warning-outline-checked" variant="outline" color="warning" checked />
      <Checkbox
        id="warning-outline-focus"
        variant="outline"
        color="warning"
        className="!ring-4 !ring-warning-6 !border-warning-9"
      />
      <Checkbox id="warning-outline-disabled" variant="outline" color="warning" disabled />
      <Checkbox
        id="warning-outline-disabled-checked"
        variant="outline"
        color="warning"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="warning-outline-dark-default" variant="outline" color="warning" />
      <Checkbox id="warning-outline-dark-checked" variant="outline" color="warning" checked />
      <Checkbox
        id="warning-outline-dark-focus"
        variant="outline"
        color="warning"
        className="!ring-4 !ring-warning-6 !border-warning-9"
      />
      <Checkbox
        id="warning-outline-dark-disabled"
        variant="outline"
        color="warning"
        disabled
      />
      <Checkbox
        id="warning-outline-dark-disabled-checked"
        variant="outline"
        color="warning"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="warning-outline-default" variant="outline" color="warning" />
            <Checkbox id="warning-outline-checked" variant="outline" color="warning" checked />
            <Checkbox
              id="warning-outline-focus"
              variant="outline"
              color="warning"
              className="!ring-4 !ring-warning-6 !border-warning-9"
            />
            <Checkbox id="warning-outline-disabled" variant="outline" color="warning" disabled />
            <Checkbox
              id="warning-outline-disabled-checked"
              variant="outline"
              color="warning"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="warning-outline-dark-default" variant="outline" color="warning" />
            <Checkbox id="warning-outline-dark-checked" variant="outline" color="warning" checked />
            <Checkbox
              id="warning-outline-dark-focus"
              variant="outline"
              color="warning"
              className="!ring-4 !ring-warning-6 !border-warning-9"
            />
            <Checkbox
              id="warning-outline-dark-disabled"
              variant="outline"
              color="warning"
              disabled
            />
            <Checkbox
              id="warning-outline-dark-disabled-checked"
              variant="outline"
              color="warning"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxWarningGhost() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="warning-ghost-default" variant="ghost" color="warning" />
      <Checkbox id="warning-ghost-checked" variant="ghost" color="warning" checked />
      <Checkbox
        id="warning-ghost-hover"
        variant="ghost"
        color="warning"
        className="!bg-warning-3"
      />
      <Checkbox
        id="warning-ghost-focus"
        variant="ghost"
        color="warning"
        className="!ring-4 !ring-warning-6 !border-warning-9"
      />
      <Checkbox id="warning-ghost-disabled" variant="ghost" color="warning" disabled />
      <Checkbox
        id="warning-ghost-disabled-checked"
        variant="ghost"
        color="warning"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="warning-ghost-dark-default" variant="ghost" color="warning" />
      <Checkbox id="warning-ghost-dark-checked" variant="ghost" color="warning" checked />
      <Checkbox
        id="warning-ghost-dark-hover"
        variant="ghost"
        color="warning"
        className="!bg-warning-3"
      />
      <Checkbox
        id="warning-ghost-dark-focus"
        variant="ghost"
        color="warning"
        className="!ring-4 !ring-warning-6 !border-warning-9"
      />
      <Checkbox id="warning-ghost-dark-disabled" variant="ghost" color="warning" disabled />
      <Checkbox
        id="warning-ghost-dark-disabled-checked"
        variant="ghost"
        color="warning"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="warning-ghost-default" variant="ghost" color="warning" />
            <Checkbox id="warning-ghost-checked" variant="ghost" color="warning" checked />
            <Checkbox
              id="warning-ghost-hover"
              variant="ghost"
              color="warning"
              className="!bg-warning-3"
            />
            <Checkbox
              id="warning-ghost-focus"
              variant="ghost"
              color="warning"
              className="!ring-4 !ring-warning-6 !border-warning-9"
            />
            <Checkbox id="warning-ghost-disabled" variant="ghost" color="warning" disabled />
            <Checkbox
              id="warning-ghost-disabled-checked"
              variant="ghost"
              color="warning"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="warning-ghost-dark-default" variant="ghost" color="warning" />
            <Checkbox id="warning-ghost-dark-checked" variant="ghost" color="warning" checked />
            <Checkbox
              id="warning-ghost-dark-hover"
              variant="ghost"
              color="warning"
              className="!bg-warning-3"
            />
            <Checkbox
              id="warning-ghost-dark-focus"
              variant="ghost"
              color="warning"
              className="!ring-4 !ring-warning-6 !border-warning-9"
            />
            <Checkbox id="warning-ghost-dark-disabled" variant="ghost" color="warning" disabled />
            <Checkbox
              id="warning-ghost-dark-disabled-checked"
              variant="ghost"
              color="warning"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxSuccessPrimary() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="success-primary-default" variant="primary" color="success" />
      <Checkbox id="success-primary-checked" variant="primary" color="success" checked />
      <Checkbox
        id="success-primary-focus"
        variant="primary"
        color="success"
        className="!ring-4 !ring-success-6"
      />
      <Checkbox id="success-primary-disabled" variant="primary" color="success" disabled />
      <Checkbox
        id="success-primary-disabled-checked"
        variant="primary"
        color="success"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="success-primary-dark-default" variant="primary" color="success" />
      <Checkbox id="success-primary-dark-checked" variant="primary" color="success" checked />
      <Checkbox
        id="success-primary-dark-focus"
        variant="primary"
        color="success"
        className="!ring-4 !ring-success-6"
      />
      <Checkbox
        id="success-primary-dark-disabled"
        variant="primary"
        color="success"
        disabled
      />
      <Checkbox
        id="success-primary-dark-disabled-checked"
        variant="primary"
        color="success"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="success-primary-default" variant="primary" color="success" />
            <Checkbox id="success-primary-checked" variant="primary" color="success" checked />
            <Checkbox
              id="success-primary-focus"
              variant="primary"
              color="success"
              className="!ring-4 !ring-success-6"
            />
            <Checkbox id="success-primary-disabled" variant="primary" color="success" disabled />
            <Checkbox
              id="success-primary-disabled-checked"
              variant="primary"
              color="success"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="success-primary-dark-default" variant="primary" color="success" />
            <Checkbox id="success-primary-dark-checked" variant="primary" color="success" checked />
            <Checkbox
              id="success-primary-dark-focus"
              variant="primary"
              color="success"
              className="!ring-4 !ring-success-6"
            />
            <Checkbox
              id="success-primary-dark-disabled"
              variant="primary"
              color="success"
              disabled
            />
            <Checkbox
              id="success-primary-dark-disabled-checked"
              variant="primary"
              color="success"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxSuccessOutline() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="success-outline-default" variant="outline" color="success" />
      <Checkbox id="success-outline-checked" variant="outline" color="success" checked />
      <Checkbox
        id="success-outline-focus"
        variant="outline"
        color="success"
        className="!ring-4 !ring-success-6 !border-success-9"
      />
      <Checkbox id="success-outline-disabled" variant="outline" color="success" disabled />
      <Checkbox
        id="success-outline-disabled-checked"
        variant="outline"
        color="success"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="success-outline-dark-default" variant="outline" color="success" />
      <Checkbox id="success-outline-dark-checked" variant="outline" color="success" checked />
      <Checkbox
        id="success-outline-dark-focus"
        variant="outline"
        color="success"
        className="!ring-4 !ring-success-6 !border-success-9"
      />
      <Checkbox
        id="success-outline-dark-disabled"
        variant="outline"
        color="success"
        disabled
      />
      <Checkbox
        id="success-outline-dark-disabled-checked"
        variant="outline"
        color="success"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="success-outline-default" variant="outline" color="success" />
            <Checkbox id="success-outline-checked" variant="outline" color="success" checked />
            <Checkbox
              id="success-outline-focus"
              variant="outline"
              color="success"
              className="!ring-4 !ring-success-6 !border-success-9"
            />
            <Checkbox id="success-outline-disabled" variant="outline" color="success" disabled />
            <Checkbox
              id="success-outline-disabled-checked"
              variant="outline"
              color="success"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="success-outline-dark-default" variant="outline" color="success" />
            <Checkbox id="success-outline-dark-checked" variant="outline" color="success" checked />
            <Checkbox
              id="success-outline-dark-focus"
              variant="outline"
              color="success"
              className="!ring-4 !ring-success-6 !border-success-9"
            />
            <Checkbox
              id="success-outline-dark-disabled"
              variant="outline"
              color="success"
              disabled
            />
            <Checkbox
              id="success-outline-dark-disabled-checked"
              variant="outline"
              color="success"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxSuccessGhost() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="success-ghost-default" variant="ghost" color="success" />
      <Checkbox id="success-ghost-checked" variant="ghost" color="success" checked />
      <Checkbox
        id="success-ghost-hover"
        variant="ghost"
        color="success"
        className="!bg-success-3"
      />
      <Checkbox
        id="success-ghost-focus"
        variant="ghost"
        color="success"
        className="!ring-4 !ring-success-6 !border-success-9"
      />
      <Checkbox id="success-ghost-disabled" variant="ghost" color="success" disabled />
      <Checkbox
        id="success-ghost-disabled-checked"
        variant="ghost"
        color="success"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="success-ghost-dark-default" variant="ghost" color="success" />
      <Checkbox id="success-ghost-dark-checked" variant="ghost" color="success" checked />
      <Checkbox
        id="success-ghost-dark-hover"
        variant="ghost"
        color="success"
        className="!bg-success-3"
      />
      <Checkbox
        id="success-ghost-dark-focus"
        variant="ghost"
        color="success"
        className="!ring-4 !ring-success-6 !border-success-9"
      />
      <Checkbox id="success-ghost-dark-disabled" variant="ghost" color="success" disabled />
      <Checkbox
        id="success-ghost-dark-disabled-checked"
        variant="ghost"
        color="success"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="success-ghost-default" variant="ghost" color="success" />
            <Checkbox id="success-ghost-checked" variant="ghost" color="success" checked />
            <Checkbox
              id="success-ghost-hover"
              variant="ghost"
              color="success"
              className="!bg-success-3"
            />
            <Checkbox
              id="success-ghost-focus"
              variant="ghost"
              color="success"
              className="!ring-4 !ring-success-6 !border-success-9"
            />
            <Checkbox id="success-ghost-disabled" variant="ghost" color="success" disabled />
            <Checkbox
              id="success-ghost-disabled-checked"
              variant="ghost"
              color="success"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="success-ghost-dark-default" variant="ghost" color="success" />
            <Checkbox id="success-ghost-dark-checked" variant="ghost" color="success" checked />
            <Checkbox
              id="success-ghost-dark-hover"
              variant="ghost"
              color="success"
              className="!bg-success-3"
            />
            <Checkbox
              id="success-ghost-dark-focus"
              variant="ghost"
              color="success"
              className="!ring-4 !ring-success-6 !border-success-9"
            />
            <Checkbox id="success-ghost-dark-disabled" variant="ghost" color="success" disabled />
            <Checkbox
              id="success-ghost-dark-disabled-checked"
              variant="ghost"
              color="success"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxInfoPrimary() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="info-primary-default" variant="primary" color="info" />
      <Checkbox id="info-primary-checked" variant="primary" color="info" checked />
      <Checkbox
        id="info-primary-focus"
        variant="primary"
        color="info"
        className="!ring-4 !ring-info-6"
      />
      <Checkbox id="info-primary-disabled" variant="primary" color="info" disabled />
      <Checkbox
        id="info-primary-disabled-checked"
        variant="primary"
        color="info"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="info-primary-dark-default" variant="primary" color="info" />
      <Checkbox id="info-primary-dark-checked" variant="primary" color="info" checked />
      <Checkbox
        id="info-primary-dark-focus"
        variant="primary"
        color="info"
        className="!ring-4 !ring-info-6"
      />
      <Checkbox id="info-primary-dark-disabled" variant="primary" color="info" disabled />
      <Checkbox
        id="info-primary-dark-disabled-checked"
        variant="primary"
        color="info"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="info-primary-default" variant="primary" color="info" />
            <Checkbox id="info-primary-checked" variant="primary" color="info" checked />
            <Checkbox
              id="info-primary-focus"
              variant="primary"
              color="info"
              className="!ring-4 !ring-info-6"
            />
            <Checkbox id="info-primary-disabled" variant="primary" color="info" disabled />
            <Checkbox
              id="info-primary-disabled-checked"
              variant="primary"
              color="info"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="info-primary-dark-default" variant="primary" color="info" />
            <Checkbox id="info-primary-dark-checked" variant="primary" color="info" checked />
            <Checkbox
              id="info-primary-dark-focus"
              variant="primary"
              color="info"
              className="!ring-4 !ring-info-6"
            />
            <Checkbox id="info-primary-dark-disabled" variant="primary" color="info" disabled />
            <Checkbox
              id="info-primary-dark-disabled-checked"
              variant="primary"
              color="info"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxInfoOutline() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="info-outline-default" variant="outline" color="info" />
      <Checkbox id="info-outline-checked" variant="outline" color="info" checked />
      <Checkbox
        id="info-outline-focus"
        variant="outline"
        color="info"
        className="!ring-4 !ring-info-6 !border-info-9"
      />
      <Checkbox id="info-outline-disabled" variant="outline" color="info" disabled />
      <Checkbox
        id="info-outline-disabled-checked"
        variant="outline"
        color="info"
        disabled
        checked
      />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="info-outline-dark-default" variant="outline" color="info" />
      <Checkbox id="info-outline-dark-checked" variant="outline" color="info" checked />
      <Checkbox
        id="info-outline-dark-focus"
        variant="outline"
        color="info"
        className="!ring-4 !ring-info-6 !border-info-9"
      />
      <Checkbox id="info-outline-dark-disabled" variant="outline" color="info" disabled />
      <Checkbox
        id="info-outline-dark-disabled-checked"
        variant="outline"
        color="info"
        disabled
        checked
      />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="info-outline-default" variant="outline" color="info" />
            <Checkbox id="info-outline-checked" variant="outline" color="info" checked />
            <Checkbox
              id="info-outline-focus"
              variant="outline"
              color="info"
              className="!ring-4 !ring-info-6 !border-info-9"
            />
            <Checkbox id="info-outline-disabled" variant="outline" color="info" disabled />
            <Checkbox
              id="info-outline-disabled-checked"
              variant="outline"
              color="info"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="info-outline-dark-default" variant="outline" color="info" />
            <Checkbox id="info-outline-dark-checked" variant="outline" color="info" checked />
            <Checkbox
              id="info-outline-dark-focus"
              variant="outline"
              color="info"
              className="!ring-4 !ring-info-6 !border-info-9"
            />
            <Checkbox id="info-outline-dark-disabled" variant="outline" color="info" disabled />
            <Checkbox
              id="info-outline-dark-disabled-checked"
              variant="outline"
              color="info"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxInfoGhost() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
<div>
  <h4 className="text-sm font-medium mb-2">Light</h4>
  <div className="flex flex-wrap items-center gap-4">
    <Checkbox id="info-ghost-default" variant="ghost" color="info" />
    <Checkbox id="info-ghost-checked" variant="ghost" color="info" checked />
    <Checkbox id="info-ghost-hover" variant="ghost" color="info" className="!bg-info-3" />
    <Checkbox
      id="info-ghost-focus"
      variant="ghost"
      color="info"
      className="!ring-4 !ring-info-6 !border-info-9"
    />
    <Checkbox id="info-ghost-disabled" variant="ghost" color="info" disabled />
    <Checkbox
      id="info-ghost-disabled-checked"
      variant="ghost"
      color="info"
      disabled
      checked
    />
  </div>
</div>

<div>
  <h4 className="text-sm font-medium mb-2">Dark</h4>
  <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
    <Checkbox id="info-ghost-dark-default" variant="ghost" color="info" />
    <Checkbox id="info-ghost-dark-checked" variant="ghost" color="info" checked />
    <Checkbox
      id="info-ghost-dark-hover"
      variant="ghost"
      color="info"
      className="!bg-info-3"
    />
    <Checkbox
      id="info-ghost-dark-focus"
      variant="ghost"
      color="info"
      className="!ring-4 !ring-info-6 !border-info-9"
    />
    <Checkbox id="info-ghost-dark-disabled" variant="ghost" color="info" disabled />
    <Checkbox
      id="info-ghost-dark-disabled-checked"
      variant="ghost"
      color="info"
      disabled
      checked
    />
  </div>
</div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="info-ghost-default" variant="ghost" color="info" />
            <Checkbox id="info-ghost-checked" variant="ghost" color="info" checked />
            <Checkbox id="info-ghost-hover" variant="ghost" color="info" className="!bg-info-3" />
            <Checkbox
              id="info-ghost-focus"
              variant="ghost"
              color="info"
              className="!ring-4 !ring-info-6 !border-info-9"
            />
            <Checkbox id="info-ghost-disabled" variant="ghost" color="info" disabled />
            <Checkbox
              id="info-ghost-disabled-checked"
              variant="ghost"
              color="info"
              disabled
              checked
            />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="info-ghost-dark-default" variant="ghost" color="info" />
            <Checkbox id="info-ghost-dark-checked" variant="ghost" color="info" checked />
            <Checkbox
              id="info-ghost-dark-hover"
              variant="ghost"
              color="info"
              className="!bg-info-3"
            />
            <Checkbox
              id="info-ghost-dark-focus"
              variant="ghost"
              color="info"
              className="!ring-4 !ring-info-6 !border-info-9"
            />
            <Checkbox id="info-ghost-dark-disabled" variant="ghost" color="info" disabled />
            <Checkbox
              id="info-ghost-dark-disabled-checked"
              variant="ghost"
              color="info"
              disabled
              checked
            />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxSizeVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-wrap items-center gap-4">
      <Checkbox id="size-sm" size="sm" checked />
      <Checkbox id="size-md" size="md" checked />
      <Checkbox id="size-lg" size="lg" checked />
      <Checkbox id="size-xlg" size="xlg" checked />
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
      <Checkbox id="size-sm-dark" size="sm" checked />
      <Checkbox id="size-md-dark" size="md" checked />
      <Checkbox id="size-lg-dark" size="lg" checked />
      <Checkbox id="size-xlg-dark" size="xlg" checked />
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-wrap items-center gap-4">
            <Checkbox id="size-sm" size="sm" checked />
            <Checkbox id="size-md" size="md" checked />
            <Checkbox id="size-lg" size="lg" checked />
            <Checkbox id="size-xlg" size="xlg" checked />
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-wrap items-center gap-4 dark">
            <Checkbox id="size-sm-dark" size="sm" checked />
            <Checkbox id="size-md-dark" size="md" checked />
            <Checkbox id="size-lg-dark" size="lg" checked />
            <Checkbox id="size-xlg-dark" size="xlg" checked />
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxWithLabels() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Light</h4>
    <div className="flex flex-col gap-2">
      <div className="flex items-center space-x-2">
        <Checkbox id="checkbox-terms" />
        <label
          htmlFor="checkbox-terms"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Accept terms and conditions
        </label>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="checkbox-disabled" disabled />
        <label
          htmlFor="checkbox-disabled"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Disabled option
        </label>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="checkbox-checked" checked />
        <label
          htmlFor="checkbox-checked"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Selected option
        </label>
      </div>
    </div>
  </div>

  <div>
    <h4 className="text-sm font-medium mb-2">Dark</h4>
    <div className="bg-black p-4 rounded-md flex flex-col gap-2 dark">
      <div className="flex items-center space-x-2">
        <Checkbox id="checkbox-terms-dark" />
        <label
          htmlFor="checkbox-terms-dark"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Accept terms and conditions
        </label>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="checkbox-disabled-dark" disabled />
        <label
          htmlFor="checkbox-disabled-dark"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Disabled option
        </label>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="checkbox-checked-dark" checked />
        <label
          htmlFor="checkbox-checked-dark"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Selected option
        </label>
      </div>
    </div>
  </div>
</div>`}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Light</h4>
          <div className="flex flex-col gap-2">
            <div className="flex items-center space-x-2">
              <Checkbox id="checkbox-terms" />
              <label
                htmlFor="checkbox-terms"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Accept terms and conditions
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="checkbox-disabled" disabled />
              <label
                htmlFor="checkbox-disabled"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Disabled option
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="checkbox-checked" checked />
              <label
                htmlFor="checkbox-checked"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Selected option
              </label>
            </div>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-medium mb-2">Dark</h4>
          <div className="bg-black p-4 rounded-md flex flex-col gap-2 dark">
            <div className="flex items-center space-x-2">
              <Checkbox id="checkbox-terms-dark" />
              <label
                htmlFor="checkbox-terms-dark"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Accept terms and conditions
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="checkbox-disabled-dark" disabled />
              <label
                htmlFor="checkbox-disabled-dark"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Disabled option
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="checkbox-checked-dark" checked />
              <label
                htmlFor="checkbox-checked-dark"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Selected option
              </label>
            </div>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CheckboxGroupExample() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`
<div className="flex flex-col gap-6">
  <div>
    <h4 className="text-sm font-medium mb-2">Notification Preferences</h4>
    <div className="space-y-3">
      <div className="flex items-center space-x-2">
        <Checkbox id="email-notifications" />
        <label
          htmlFor="email-notifications"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Email notifications
        </label>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="sms-notifications" />
        <label
          htmlFor="sms-notifications"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          SMS notifications
        </label>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="push-notifications" />
        <label
          htmlFor="push-notifications"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          Push notifications
        </label>
      </div>
    </div>
  </div>
</div>
    `}
    >
      <div className="flex flex-col gap-6">
        <div>
          <h4 className="text-sm font-medium mb-2">Notification Preferences</h4>
          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox id="email-notifications" />
              <label
                htmlFor="email-notifications"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Email notifications
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="sms-notifications" />
              <label
                htmlFor="sms-notifications"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                SMS notifications
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="push-notifications" />
              <label
                htmlFor="push-notifications"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Push notifications
              </label>
            </div>
          </div>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
