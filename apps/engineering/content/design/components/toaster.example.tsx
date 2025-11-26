"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button, toast } from "@unkey/ui";

export function BasicToast() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <Button onClick={() => toast("This is a basic toast notification")}>
    Show Basic Toast
  </Button>
  <p className="text-sm text-gray-600">
    Click the button above to trigger a basic toast notification
  </p>
</div>`}
    >
      <div className="space-y-4">
        <Button onClick={() => toast("This is a basic toast notification")}>
          Show Basic Toast
        </Button>
        <p className="text-sm text-gray-600">
          Click the button above to trigger a basic toast notification
        </p>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ToastVariants() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <div className="flex flex-wrap gap-2">
    <Button onClick={() => toast.success("Success! Your action was completed.")}>
      Success Toast
    </Button>
    <Button onClick={() => toast.error("Error! Something went wrong.")}>Error Toast</Button>
    <Button onClick={() => toast.warning("Warning! Please check your input.")}>
      Warning Toast
    </Button>
    <Button onClick={() => toast.info("Info: Here's some helpful information.")}>
      Info Toast
    </Button>
  </div>
  <p className="text-sm text-gray-600">Different toast types for different scenarios</p>
</div>`}
    >
      <div className="space-y-4">
        <div className="flex flex-wrap gap-2">
          <Button onClick={() => toast.success("Success! Your action was completed.")}>
            Success Toast
          </Button>
          <Button onClick={() => toast.error("Error! Something went wrong.")}>Error Toast</Button>
          <Button onClick={() => toast.warning("Warning! Please check your input.")}>
            Warning Toast
          </Button>
          <Button onClick={() => toast.info("Info: Here's some helpful information.")}>
            Info Toast
          </Button>
        </div>
        <p className="text-sm text-gray-600">Different toast types for different scenarios</p>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ToastWithDescription() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <Button
    onClick={() =>
      toast("User Updated", {
        description:
          "The user profile has been successfully updated with the new information.",
      })
    }
  >
    Toast with Description
  </Button>
  <p className="text-sm text-gray-600">
    Toasts can include both a title and description for more detailed information
  </p>
</div>`}
    >
      <div className="space-y-4">
        <Button
          onClick={() =>
            toast("User Updated", {
              description:
                "The user profile has been successfully updated with the new information.",
            })
          }
        >
          Toast with Description
        </Button>
        <p className="text-sm text-gray-600">
          Toasts can include both a title and description for more detailed information
        </p>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ToastWithActions() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <div className="flex flex-wrap gap-2">
    <Button
      onClick={() =>
        toast("Undo Changes", {
          description: "Your changes have been saved.",
          action: {
            label: "Undo",
            onClick: () => console.log("Undo clicked"),
          },
        })
      }
    >
      Toast with Action
    </Button>
    <Button
      onClick={() =>
        toast("Confirm Deletion", {
          description: "Are you sure you want to delete this item?",
          action: {
            label: "Delete",
            onClick: () => console.log("Delete confirmed"),
          },
          cancel: {
            label: "Cancel",
            onClick: () => console.log("Delete cancelled"),
          },
        })
      }
      variant="destructive"
    >
      Toast with Action & Cancel
    </Button>
  </div>
  <p className="text-sm text-gray-600">
    Toasts can include action buttons for user interaction
  </p>
</div>`}
    >
      <div className="space-y-4">
        <div className="flex flex-wrap gap-2">
          <Button
            onClick={() =>
              toast("Undo Changes", {
                description: "Your changes have been saved.",
                action: {
                  label: "Undo",
                  onClick: () => {},
                },
              })
            }
          >
            Toast with Action
          </Button>
          <Button
            onClick={() =>
              toast("Confirm Deletion", {
                description: "Are you sure you want to delete this item?",
                action: {
                  label: "Delete",
                  onClick: () => {},
                },
                cancel: {
                  label: "Cancel",
                  onClick: () => {},
                },
              })
            }
            variant="destructive"
          >
            Toast with Action & Cancel
          </Button>
        </div>
        <p className="text-sm text-gray-600">
          Toasts can include action buttons for user interaction
        </p>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ToastWithCustomDuration() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <div className="flex flex-wrap gap-2">
    <Button
      onClick={() =>
        toast("Persistent Message", {
          duration: Number.POSITIVE_INFINITY, // Never auto-dismiss
          closeButton: true,
        })
      }
    >
      Persistent Toast
    </Button>
  </div>
  <p className="text-sm text-gray-600">
    Control how long toasts stay visible with the duration option
  </p>
</div>`}
    >
      <div className="space-y-4">
        <div className="flex flex-wrap gap-2">
          <Button
            onClick={() =>
              toast("Persistent Message", {
                duration: Number.POSITIVE_INFINITY, // Never auto-dismiss
                closeButton: true,
              })
            }
          >
            Persistent Toast
          </Button>
        </div>
        <p className="text-sm text-gray-600">
          Control how long toasts stay visible with the duration option
        </p>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ToastWithPromise() {
  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<div className="space-y-4">
  <Button
    onClick={() => {
      toast.promise(new Promise((resolve) => setTimeout(resolve, 2000)), {
        loading: "Saving your changes...",
        success: "Changes saved successfully!",
        error: "Failed to save changes",
      });
    }}
  >
    Promise Toast
  </Button>
  <p className="text-sm text-gray-600">
    Show loading, success, and error states for async operations
  </p>
</div>`}
    >
      <div className="space-y-4">
        <Button
          onClick={() => {
            toast.promise(new Promise((resolve) => setTimeout(resolve, 2000)), {
              loading: "Saving your changes...",
              success: "Changes saved successfully!",
              error: "Failed to save changes",
            });
          }}
        >
          Promise Toast
        </Button>
        <p className="text-sm text-gray-600">
          Show loading, success, and error states for async operations
        </p>
      </div>
    </RenderComponentWithSnippet>
  );
}
