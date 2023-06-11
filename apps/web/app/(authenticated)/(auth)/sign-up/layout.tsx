export default function AuthLayout(props: { children: React.ReactNode }) {
  return (
    <>
      <div className="relative grid min-h-screen grid-cols-1 overflow-hidden md:grid-cols-3 lg:grid-cols-2">
        <div className="relative">
          <div
            className="absolute inset-0 bg-cover">
          </div>
        </div>
        <div className="absolute inset-0 bg-gradient-to-t from-background to-background/60 md:hidden" />
        <div className="container absolute top-1/2 col-span-1 flex -translate-y-1/2 items-center md:static md:top-0 md:col-span-2 md:flex md:translate-y-0 lg:col-span-1">
          {props.children}
        </div>
      </div >
    </>
  );
}
