import Link from 'next/link'
import { addEmail } from '@/app/actions/addEmail'
import { Container } from '@/components/landingComponents/Container'
import { FadeIn } from '@/components/landingComponents/FadeIn'
import { socialMediaProfiles } from '@/components/landingComponents/SocialMedia'
import { experimental_useFormStatus as useFormStatus } from 'react-dom'
import { useToast } from '../ui/use-toast'
const navigation = [
  {
    title: 'Company',
    links: [
      { title: 'About', href: '/about' },
      { title: 'Blog', href: '/blog' },
      { title: 'Changelog', href: '/changelog' },
      { title: 'Analytics', href: 'https://plausible.io/unkey.dev' },
      { title: 'Unkey', href: 'https://github.com/unkeyed/unkey' },
    ],
  },
  {
    title: 'Legal',
    links: [
      { title: 'Privacy Policy', href: '/policies/privacy' },
      { title: 'Terms', href: '/policies/terms' },
    ],
  },
  {
    title: 'Connect',
    links: socialMediaProfiles,
  },
]

function Navigation() {
  return (
    <nav>
      <ul role="list" className="grid grid-cols-2 gap-8 sm:grid-cols-3">
        {navigation.map((section) => (
          <li key={section.title}>
            <div className="font-display text-sm font-semibold tracking-wider text-neutral-950">
              {section.title}
            </div>
            <ul role="list" className="mt-4 text-sm text-neutral-700">
              {section.links.map((link) => (
                <li key={link.title} className="mt-4">
                  <Link
                    href={link.href}
                    className="transition hover:text-neutral-950"
                  >
                    {link.title}
                  </Link>
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </nav>
  )
}

function ArrowIcon(props : any) {
  return (
    <svg viewBox="0 0 16 6" aria-hidden="true" {...props}>
      <path
        fill="currentColor"
        fillRule="evenodd"
        clipRule="evenodd"
        d="M16 3 10 .5v2H0v1h10v2L16 3Z"
      />
    </svg>
  )
}

function NewsletterForm() {
  const { pending } = useFormStatus()
  const {toast} = useToast()
  return (
    <form className="max-w-md" action={async (data: FormData) => {
      const email = data.get("email")
      if (!email) {
        toast({
          title: "Error",
          description: "Please enter an email",
          variant: "destructive"
        })
        return
      }
      const res = await addEmail(email as string);
      if (res.success === true) {
        data.set("email", "") 
        toast({
          title: "Success",
          description: "Thanks for signing up!",
          variant: "default"
        })
        return;
      } else {
        toast({
          title: "Error",
          description: "Something went wrong",
          variant: "destructive"
        })
      }
    }}>
      <h2 className="font-display text-sm font-semibold tracking-wider text-neutral-950">
        Sign up for our newsletter
      </h2>
      <p className="mt-4 text-sm text-neutral-700">
        Subscribe to get the latest Unkey news
      </p>
      <div className="relative mt-6">
        <input
          type="email"
          name="email"
          id='email'
          placeholder="Email address"
          autoComplete="email"
          aria-label="Email address"
          className="block w-full rounded-2xl border border-neutral-300 bg-transparent py-4 pl-6 pr-20 text-base/6 text-neutral-950 ring-4 ring-transparent transition placeholder:text-neutral-500 focus:border-neutral-950 focus:outline-none focus:ring-neutral-950/5"
        />
        <div className="absolute inset-y-1 right-1 flex justify-end">
          <button
            type="submit"
            aria-label="Submit"
            disabled={pending}
            className="flex aspect-square h-full items-center justify-center rounded-xl bg-neutral-950 text-white transition hover:bg-neutral-800"
          >
            <ArrowIcon className="w-4" />
          </button>
        </div>
      </div>
    </form>
  )
}

export function Footer() {
  return (
    <Container as="footer" className="mt-24 w-full sm:mt-32 lg:mt-40">
      <FadeIn>
        <div className="grid grid-cols-1 gap-x-8 gap-y-16 lg:grid-cols-2">
          <Navigation />
          <div className="flex lg:justify-end">
            <NewsletterForm />
          </div>
        </div>
        <div className="mb-20 mt-24 flex flex-wrap items-end justify-between gap-x-6 gap-y-4 border-t border-neutral-950/10 pt-12">
          <Link href="/" aria-label="Home">
            <svg width="32" height="32" viewBox="0 0 276 276" fill="none" xmlns="http://www.w3.org/2000/svg">
              <g filter="url(#filter0_d_101_3)">
                <path d="M160.206 70H197V156.749C197 167.064 194.529 175.99 189.588 183.528C184.691 191.021 177.853 196.818 169.074 200.917C160.294 204.972 150.103 207 138.5 207C126.809 207 116.574 204.972 107.794 200.917C99.0147 196.818 92.1765 191.021 87.2794 183.528C82.4265 175.99 80 167.064 80 156.749V70H116.794V153.575C116.794 157.763 117.721 161.51 119.574 164.816C121.426 168.078 123.985 170.634 127.25 172.486C130.559 174.337 134.309 175.263 138.5 175.263C142.735 175.263 146.485 174.337 149.75 172.486C153.015 170.634 155.574 168.078 157.426 164.816C159.279 161.51 160.206 157.763 160.206 153.575V70Z" fill="url(#paint0_linear_101_3)" shapeRendering="crispEdges" />
                <path d="M160.206 69.5H159.706V70V153.575C159.706 157.686 158.797 161.346 156.991 164.57C155.183 167.753 152.689 170.244 149.503 172.051C146.323 173.854 142.66 174.763 138.5 174.763C134.386 174.763 130.722 173.855 127.496 172.05C124.311 170.244 121.817 167.753 120.009 164.57C118.203 161.346 117.294 157.686 117.294 153.575V70V69.5H116.794H80H79.5V70V156.749C79.5 167.145 81.9466 176.168 86.859 183.798L86.8609 183.801C91.813 191.379 98.726 197.235 107.583 201.37L107.584 201.371C116.442 205.462 126.751 207.5 138.5 207.5C150.161 207.5 160.426 205.462 169.283 201.371L169.285 201.37C178.141 197.235 185.054 191.379 190.006 183.802C195.008 176.171 197.5 167.147 197.5 156.749V70V69.5H197H160.206Z" stroke="url(#paint1_linear_101_3)" shapeRendering="crispEdges" />
              </g>
              <defs>
                <filter id="filter0_d_101_3" x="75" y="69" width="127" height="147" filterUnits="userSpaceOnUse" colorInterpolationFilters="sRGB">
                  <feFlood floodOpacity="0" result="BackgroundImageFix" />
                  <feColorMatrix in="SourceAlpha" type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" result="hardAlpha" />
                  <feOffset dy="4" />
                  <feGaussianBlur stdDeviation="2" />
                  <feComposite in2="hardAlpha" operator="out" />
                  <feColorMatrix type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0" />
                  <feBlend mode="normal" in2="BackgroundImageFix" result="effect1_dropShadow_101_3" />
                  <feBlend mode="normal" in="SourceGraphic" in2="effect1_dropShadow_101_3" result="shape" />
                </filter>
                <linearGradient id="paint0_linear_101_3" x1="80" y1="70" x2="176.419" y2="207.057" gradientUnits="userSpaceOnUse">
                  <stop offset="0.161458" />
                  <stop offset="1" stopColor="#B6B6B6" stopOpacity="0" />
                </linearGradient>
                <linearGradient id="paint1_linear_101_3" x1="47.5" y1="168.5" x2="212.999" y2="167.862" gradientUnits="userSpaceOnUse">
                  <stop offset="0.194498" />
                  <stop offset="0.411458" stopColor="white" stopOpacity="0" />
                </linearGradient>
              </defs>
            </svg>
          </Link>
          <p className="text-sm text-neutral-700">
            © Unkeyed, Inc. {new Date().getFullYear()}
          </p>
        </div>
      </FadeIn>
    </Container>
  )
}
