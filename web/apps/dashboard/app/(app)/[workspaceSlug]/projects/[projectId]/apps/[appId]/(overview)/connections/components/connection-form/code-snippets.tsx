import { CopyButton, SegmentedControl, SegmentedControlItem } from "@unkey/ui";
import { useState } from "react";

type Language = "node" | "go" | "python";

const LANGUAGES: Record<Language, { label: string; snippet: (variable: string) => string }> = {
  node: { label: "Node.js", snippet: (variable) => `const host = process.env.${variable};` },
  go: { label: "Go", snippet: (variable) => `host := os.Getenv("${variable}")` },
  python: { label: "Python", snippet: (variable) => `host = os.environ["${variable}"]` },
};

const LANGUAGE_ORDER: Language[] = ["node", "go", "python"];

export function CodeSnippets({
  variable,
  targetAppName,
}: { variable: string; targetAppName: string }) {
  const [language, setLanguage] = useState<Language>("node");
  const snippet = LANGUAGES[language].snippet(variable);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-gray-12">Use it in your code</span>
        <SegmentedControl<Language>
          aria-label="Language"
          value={language}
          onValueChange={setLanguage}
        >
          {LANGUAGE_ORDER.map((value) => (
            <SegmentedControlItem<Language> key={value} value={value} className="text-xs">
              {LANGUAGES[value].label}
            </SegmentedControlItem>
          ))}
        </SegmentedControl>
      </div>
      <div className="flex min-w-0 items-center gap-2 rounded-md bg-background px-2 py-1.5 ring-1 ring-grayA-4">
        <code className="min-w-0 flex-1 break-all font-mono text-xs leading-5 text-gray-12">
          {snippet}
        </code>
        <CopyButton value={snippet} variant="ghost" className="shrink-0" />
      </div>
      <p className="text-xs text-gray-11">Add the protocol and port {targetAppName} listens on.</p>
    </div>
  );
}
