import { RenderComponentWithSnippet } from "@/app/components/render";
import { CircleProgress } from "@unkey/ui";

export function BasicProgress() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-6 justify-center items-center">
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={0} total={5} />
        <span className="text-xs text-gray-500">0/5 - Starting</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={2} total={5} />
        <span className="text-xs text-gray-500">2/5 - In Progress</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={4} total={5} />
        <span className="text-xs text-gray-500">4/5 - Almost Done</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={5} total={5} />
        <span className="text-xs text-gray-500">5/5 - Complete</span>
    </div>
</div>`}
    >
      <div className="flex flex-wrap gap-6 justify-center items-center">
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={0} total={5} />
          <span className="text-xs text-gray-500">0/5 - Starting</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={2} total={5} />
          <span className="text-xs text-gray-500">2/5 - In Progress</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={4} total={5} />
          <span className="text-xs text-gray-500">4/5 - Almost Done</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={5} total={5} />
          <span className="text-xs text-gray-500">5/5 - Complete</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function VariantExamples() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-6 justify-center items-center">
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} variant="primary" />
        <span className="text-xs text-gray-500">Primary</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} variant="secondary" />
        <span className="text-xs text-gray-500">Secondary</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={4} total={5} variant="success" />
        <span className="text-xs text-gray-500">Success</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} variant="warning" />
        <span className="text-xs text-gray-500">Warning</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={2} total={5} variant="error" />
        <span className="text-xs text-gray-500">Error</span>
    </div>
</div>`}
    >
      <div className="flex flex-wrap gap-6 justify-center items-center">
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} variant="primary" />
          <span className="text-xs text-gray-500">Primary</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} variant="secondary" />
          <span className="text-xs text-gray-500">Secondary</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={4} total={5} variant="success" />
          <span className="text-xs text-gray-500">Success</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} variant="warning" />
          <span className="text-xs text-gray-500">Warning</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={2} total={5} variant="error" />
          <span className="text-xs text-gray-500">Error</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function SizeExamples() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-6 justify-center items-center">
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} size="sm-thin" />
        <span className="text-xs text-gray-500">Small Thin</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} size="sm-regular" />
        <span className="text-xs text-gray-500">Small Regular</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} size="md-regular" />
        <span className="text-xs text-gray-500">Medium Regular</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} size="lg-medium" />
        <span className="text-xs text-gray-500">Large Medium</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} size="xl-bold" />
        <span className="text-xs text-gray-500">XL Bold</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={5} size="2xl-bold" />
        <span className="text-xs text-gray-500">2XL Bold</span>
    </div>
</div>`}
    >
      <div className="flex flex-wrap gap-6 justify-center items-center">
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} size="sm-thin" />
          <span className="text-xs text-gray-500">Small Thin</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} size="sm-regular" />
          <span className="text-xs text-gray-500">Small Regular</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} size="md-regular" />
          <span className="text-xs text-gray-500">Medium Regular</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} size="lg-medium" />
          <span className="text-xs text-gray-500">Large Medium</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} size="xl-bold" />
          <span className="text-xs text-gray-500">XL Bold</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={5} size="2xl-bold" />
          <span className="text-xs text-gray-500">2XL Bold</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CompletionStates() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-6 justify-center items-center">
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={25} total={100} variant="primary" />
        <span className="text-xs text-gray-500">25% Complete</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={50} total={100} variant="primary" />
        <span className="text-xs text-gray-500">50% Complete</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={75} total={100} variant="primary" />
        <span className="text-xs text-gray-500">75% Complete</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={100} total={100} variant="success" />
        <span className="text-xs text-gray-500">✓ Complete</span>
    </div>
</div>`}
    >
      <div className="flex flex-wrap gap-6 justify-center items-center">
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={25} total={100} variant="primary" />
          <span className="text-xs text-gray-500">25% Complete</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={50} total={100} variant="primary" />
          <span className="text-xs text-gray-500">50% Complete</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={75} total={100} variant="primary" />
          <span className="text-xs text-gray-500">75% Complete</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={100} total={100} variant="success" />
          <span className="text-xs text-gray-500">✓ Complete</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function FormValidation() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-6 max-w-md">
    <div className="flex items-center gap-3">
        <CircleProgress value={0} total={4} variant="secondary" size="sm-regular" />
        <span className="text-sm">Profile Setup</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={2} total={4} variant="primary" size="sm-regular" />
        <span className="text-sm">Contact Information</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={3} total={4} variant="warning" size="sm-regular" />
        <span className="text-sm">Security Settings</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={4} total={4} variant="success" size="sm-regular" />
        <span className="text-sm">Account Verification</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={1} total={4} variant="error" size="sm-regular" />
        <span className="text-sm">Payment Details</span>
    </div>
</div>`}
    >
      <div className="space-y-6 max-w-md">
        <div className="flex items-center gap-3">
          <CircleProgress value={0} total={4} variant="secondary" size="sm-regular" />
          <span className="text-sm">Profile Setup</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={2} total={4} variant="primary" size="sm-regular" />
          <span className="text-sm">Contact Information</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={3} total={4} variant="warning" size="sm-regular" />
          <span className="text-sm">Security Settings</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={4} total={4} variant="success" size="sm-regular" />
          <span className="text-sm">Account Verification</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={1} total={4} variant="error" size="sm-regular" />
          <span className="text-sm">Payment Details</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function TaskProgress() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="flex flex-wrap gap-6 justify-center items-center">
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={8} total={10} variant="primary" size="lg-medium" />
        <span className="text-xs text-gray-500">Daily Tasks</span>
        <span className="text-xs text-gray-400">8/10 done</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={12} total={12} variant="success" size="lg-medium" />
        <span className="text-xs text-gray-500">Sprint Goals</span>
        <span className="text-xs text-gray-400">All complete</span>
    </div>
    <div className="flex flex-col gap-2 items-center">
        <CircleProgress value={3} total={7} variant="warning" size="lg-medium" />
        <span className="text-xs text-gray-500">Code Reviews</span>
        <span className="text-xs text-gray-400">3/7 done</span>
    </div>
</div>`}
    >
      <div className="flex flex-wrap gap-6 justify-center items-center">
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={8} total={10} variant="primary" size="lg-medium" />
          <span className="text-xs text-gray-500">Daily Tasks</span>
          <span className="text-xs text-gray-400">8/10 done</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={12} total={12} variant="success" size="lg-medium" />
          <span className="text-xs text-gray-500">Sprint Goals</span>
          <span className="text-xs text-gray-400">All complete</span>
        </div>
        <div className="flex flex-col gap-2 items-center">
          <CircleProgress value={3} total={7} variant="warning" size="lg-medium" />
          <span className="text-xs text-gray-500">Code Reviews</span>
          <span className="text-xs text-gray-400">3/7 done</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function LoadingStates() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-6">
    <div className="flex items-center gap-3">
        <CircleProgress value={147} total={500} variant="primary" />
        <span className="text-sm">Uploading files... 147/500</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={89} total={100} variant="secondary" />
        <span className="text-sm">Processing data... 89%</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={100} total={100} variant="success" />
        <span className="text-sm">Backup complete ✓</span>
    </div>
    <div className="flex items-center gap-3">
        <CircleProgress value={23} total={100} variant="error" />
        <span className="text-sm">Sync failed at 23%</span>
    </div>
</div>`}
    >
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <CircleProgress value={147} total={500} variant="primary" />
          <span className="text-sm">Uploading files... 147/500</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={89} total={100} variant="secondary" />
          <span className="text-sm">Processing data... 89%</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={100} total={100} variant="success" />
          <span className="text-sm">Backup complete ✓</span>
        </div>
        <div className="flex items-center gap-3">
          <CircleProgress value={23} total={100} variant="error" />
          <span className="text-sm">Sync failed at 23%</span>
        </div>
      </div>
    </RenderComponentWithSnippet>
  );
}
