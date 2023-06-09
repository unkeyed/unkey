import './globals.css'
import { Inter } from 'next/font/google'

const inter = Inter({ subsets: ['latin'] })

export const metadata = {
  title: 'Open Source API Key Management',
  description: 'Accelerate your API development',
  openGraph: {
    title: "Open Source API Key Management",
    description: 'Accelerate your API development',
    url: "https://unkey.dev",
    siteName: "unkey.dev",
    images: [
      {
        url: "https://unkey.dev/og.png",
        width: 1200,
        height: 675,
      }
    ]
  },
  twitter: {
    title: "Unkey",
    card: "summary_large_image",
  },
  icons: {
    shortcut: "/og.png",
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <head>
        <script
          src="https://beamanalytics.b-cdn.net/beam.min.js"
          data-token="e08d63f1-631b-4eeb-89b3-7799fedde74f"
          async
        />
      </head>
      <body className={`${inter.className} bg-black `}>{children}</body>
    </html>
  )
}
