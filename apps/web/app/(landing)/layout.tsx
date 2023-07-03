import { NavigationBar } from "@/components/Navbar";

export default function LandingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="bg-gray-50">
      <NavigationBar />
      {children}
    </div>
  );
}
