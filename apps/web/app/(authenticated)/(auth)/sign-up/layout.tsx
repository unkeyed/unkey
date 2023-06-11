import Link from "next/link";
import { Particles } from "@/components/particles";
export default function AuthLayout(props: { children: React.ReactNode }) {

  const infoItems = [
    {
      title: 'Save development time',
      content:
        'Add authentication and user management to your app with just a few lines of code',
    },
    {
      title: 'Increase engagement',
      content: 'Add intuitive UIs designed to decrease friction for your users',
    },
    {
      title: 'Protect your users',
      content:
        'Enable features like two-step verification and enjoy automatic security updates',
    },
    {
      title: 'Match your brand',
      content:
        'Theme our pre-built components, or integrate with our easy-to-use APIs',
    },
  ];

  return (
    <>
      <div className="relative grid min-h-screen grid-cols-1 overflow-hidden md:grid-cols-3 lg:grid-cols-2">
        <div className="relative">
          <div
            className="absolute inset-0 bg-cover">
            <Particles
              className="absolute inset-0 -z-10 "
              vy={-1}
              quantity={200}
              staticity={500}
              color="#7c3aed"
            />

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
