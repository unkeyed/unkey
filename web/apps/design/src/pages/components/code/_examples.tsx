import { Button, Code } from "@unkey/ui";
import { Preview } from "../../../components/Preview";

export function WithCopyButtonExample() {
  return (
    <Preview>
      <Code
        copyButton={
          <Button size="sm" variant="ghost">
            Copy
          </Button>
        }
      >
        <span>unkey_3ZaM7...c9Vd</span>
      </Code>
    </Preview>
  );
}
