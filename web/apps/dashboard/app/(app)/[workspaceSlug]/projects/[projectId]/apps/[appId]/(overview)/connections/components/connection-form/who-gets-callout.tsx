import { IconCircleInfoOutline18 } from "@unkey/icons";
import { Alert, AlertDescription, AlertTitle } from "@unkey/ui";
import type { WhoGetsLine } from "../../model/who-gets";

export function WhoGetsCallout({
  variable,
  lines,
}: {
  variable: string;
  lines: readonly WhoGetsLine[];
}) {
  return (
    <Alert role="note">
      <IconCircleInfoOutline18 className="size-4 text-gray-11" />
      <AlertTitle className="text-sm">
        Who gets <code className="font-mono text-xs">{variable}</code>
      </AlertTitle>
      <AlertDescription>
        <ul className="flex flex-col gap-1 text-xs text-gray-11">
          {lines.map((line) => (
            <li key={line.subject}>
              <span className="font-medium text-gray-12">{line.subject}</span> {line.rest}
            </li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
}
