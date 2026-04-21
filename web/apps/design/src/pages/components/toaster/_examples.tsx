import { Button, Toaster, toast } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function DefaultExample() {
  return (
    <Preview>
      <Toaster />
      <Button
        variant="outline"
        onClick={() => toast("Saved changes to ACME production")}
      >
        Show toast
      </Button>
    </Preview>
  );
}

export function SuccessExample() {
  return (
    <Preview>
      <Toaster />
      <Button
        variant="primary"
        color="success"
        onClick={() => toast.success("Key rotated successfully")}
      >
        Rotate key
      </Button>
    </Preview>
  );
}

export function ErrorExample() {
  return (
    <Preview>
      <Toaster />
      <Button
        variant="primary"
        color="danger"
        onClick={() =>
          toast.error("Could not reach auth provider", {
            description: "Please try again in a few minutes.",
          })
        }
      >
        Trigger error
      </Button>
    </Preview>
  );
}

export function DescriptionExample() {
  return (
    <Preview>
      <Toaster />
      <Button
        variant="outline"
        onClick={() =>
          toast("Invitation sent", {
            description: "andreas@acme.com will receive an email shortly.",
          })
        }
      >
        Invite teammate
      </Button>
    </Preview>
  );
}

export function ActionExample() {
  return (
    <Preview>
      <Toaster />
      <Button
        variant="outline"
        onClick={() =>
          toast("Deleted 3 API keys", {
            description: "You can restore them for the next 30 seconds.",
            action: {
              label: "Undo",
              onClick: () => toast.success("Restored 3 API keys"),
            },
          })
        }
      >
        Delete keys
      </Button>
    </Preview>
  );
}
