import { cn } from "@/lib/utils";

type Props = {
  text: string;
  className?: string;
};

export function TruncatedCell({ text, className }: Props) {
  return (
    <div
      className={cn(
        "whitespace-pre-wrap font-mono text-xs break-all text-pretty max-w-125 my-2",
        className,
      )}
    >
      {text}
    </div>
  );
}
