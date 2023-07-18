import { RootLayout } from '@/components/landingComponents/RootLayout'

import '@/styles/landing/styles/tailwind.css'

export const metadata = {
  title: {
    template: '%s - Unkey',
    default: 'Unkey - API management made easy',
  },
}

export default function Layout({ children } : {
  children: React.ReactNode,
}) {
  return (
    <html lang="en" className="h-full bg-neutral-950 text-base antialiased">
      <body className="flex min-h-full flex-col">
        <RootLayout>{children}</RootLayout>
      </body>
    </html>
  )
}
