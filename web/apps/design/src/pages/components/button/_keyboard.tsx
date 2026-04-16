import { Button } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function KeyboardExample() {
  return (
    <Preview>
      <Button
        keyboard={{
          display: "⌘K",
          trigger: (e) =>
            (e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k",
          callback: () => alert("⌘K pressed"),
        }}
      >
        Search
      </Button>
    </Preview>
  );
}
