
export default function LandingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (

    <div className="bg-gray-50 flex justify-center h-full">
        {children}
    </div>
  );
}
