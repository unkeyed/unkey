import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";

const environments = [
  { value: "production", label: "production" },
  { value: "preview", label: "preview" },
];

export default function SelectSizes() {
  return (
    <div className="flex w-56 flex-col gap-3">
      {(["md", "sm"] as const).map((size) => (
        <Select key={size} defaultValue="production" items={environments}>
          <SelectTrigger size={size} aria-label={`Environment, ${size}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {environments.map((env) => (
              <SelectItem key={env.value} value={env.value}>
                {env.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ))}
    </div>
  );
}
