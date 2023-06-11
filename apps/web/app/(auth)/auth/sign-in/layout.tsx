export default function AuthLayout(props: { children: React.ReactNode }) {

  return (
    <>
      <div className="grid h-screen place-items-center">
        <div className="w-full">{props.children}</div>
      </div>
    </>
  );
}
