import { Navbar } from "@/components/dashboard/navbar";

export default async function SemanticCacheLayout({ children }: { children: React.ReactNode }) {
  const navigation = [
    {
      label: "Logs",
      href: "/semantic-cache/logs",
      segment: "logs",
    },
    {
      label: "Analytics",
      href: "/semantic-cache/analytics",
      segment: "analytics",
    },
    {
      label: "Settings",
      href: "/semantic-cache/settings",
      segment: "settings",
    },
  ];

  return (
    <>
      <Navbar navigation={navigation} />
      <div>{children}</div>
    </>
  );
}
