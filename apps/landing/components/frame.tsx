import clsx from "clsx";

export function Frame({
  as: Component = "div",
  className,
  children,
}: {
  as?: any;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <Component
      className={clsx(
        "flex max-w-7xl w-fit mx-auto rounded-[30.5px] bg-gradient-to-b from-white/6 to-white/8 p-[.75px] overflow-hidden",
        className,
      )}
    >
      <div className="bg-[radial-gradient(57.42%_100%_at_100%_0%,rgba(255,255,255,0.25)_0%,rgba(255,255,255,0.2)_50%,rgba(255,255,255,0.1)_100%)] p-2">
        <div className="rounded-[24px] overflow-hidden bg-gradient-to-b from-white/20 to-white/15 p-[.75px]">
          {children}
        </div>
      </div>
    </Component>
  );
}
