import { NavigationBar } from "@/components/nav-bar";

export default function LandingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-gray-50">
      <NavigationBar />
      {children}
    </div>
  );
}
