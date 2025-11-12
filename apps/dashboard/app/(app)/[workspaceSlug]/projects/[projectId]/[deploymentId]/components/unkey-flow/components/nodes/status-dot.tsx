import { cn } from "@unkey/ui/src/lib/utils";

const colorMap = {
  success: {
    bg: "bg-success-9",
    ring: "hsl(var(--successA-4))",
  },
  info: {
    bg: "bg-info-9",
    ring: "hsl(var(--infoA-4))",
  },
} as const;

type StatusDotProps = {
  variant: keyof typeof colorMap;
};

export const StatusDot = ({ variant }: StatusDotProps) => {
  const { bg, ring } = colorMap[variant];
  return (
    <>
      <div className="absolute top-1.5 right-1.5 size-[7px]">
        {/* Ring 1 */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite",
            boxShadow: `0 0 0 1.5px ${ring}`,
          }}
        />
        {/* Ring 2 */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite 1s",
            boxShadow: `0 0 0 1.5px ${ring}`,
          }}
        />
        {/* Solid dot */}
        <div className={cn("absolute inset-0 rounded-full", bg)} />
      </div>
      <style>{`
        @keyframes breathe-ring {
          0%, 100% {
            transform: scale(1);
            opacity: 0.6;
          }
          50% {
            transform: scale(2.2);
            opacity: 0;
          }
        }
      `}</style>
    </>
  );
};
