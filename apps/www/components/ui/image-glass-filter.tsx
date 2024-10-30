export function FrostedGlassFilter({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative overflow-hidden">
      <div className="absolute top-0 right-[-60px] backdrop-blur-xl backdrop-opacity-80 placeholder-parallelogram " />
      {children}
    </div>
  );
}
