function CloudflareIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 65 32"
      fill="currentColor"
      aria-label="Cloudflare logomark"
    >
      <path d="M45.234 24.397l.517-1.808c.345-1.193.19-2.29-.434-3.093-.58-.748-1.503-1.173-2.6-1.228l-18.238-.248a.37.37 0 01-.31-.18.4.4 0 01-.034-.37c.069-.165.228-.275.407-.283l18.393-.248c2.697-.138 5.624-2.345 6.634-5.017l1.276-3.38a.63.63 0 00.034-.275C49.019 3.938 44.826 0 39.725 0c-4.47 0-8.277 2.842-9.722 6.82-.89-.675-2.014-1.076-3.228-1.02-2.207.103-3.978 1.89-4.296 4.098-.076.51-.07 1.007.014 1.476C17.907 11.553 14 15.614 14 20.573c0 .648.062 1.283.165 1.904a.62.62 0 00.607.524l29.82.007c.2-.014.386-.138.462-.324l.18-.29z" />
      <path d="M49.124 11.374a.49.49 0 00-.476.048.479.479 0 00-.207.4l-.276 1.724c-.345 1.193-.19 2.29.434 3.093.58.748 1.503 1.173 2.6 1.228l3.89.248c.172.014.324.103.4.234a.4.4 0 01.034.37c-.069.165-.228.275-.407.283l-4.048.248c-2.704.138-5.631 2.345-6.641 5.017l-.358.952a.26.26 0 00.234.358h13.107a.55.55 0 00.524-.386A11.425 11.425 0 0060 20.573c0-5.117-3.345-9.447-7.959-10.924a5.506 5.506 0 00-2.917 1.724z" />
    </svg>
  );
}

function VercelIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 74 64" fill="currentColor" aria-label="Vercel logomark">
      <path d="M37.5896 0.25L74.5396 64.25H0.639648L37.5896 0.25Z" />
    </svg>
  );
}

const providerIcons: Partial<Record<string, (props: { className?: string }) => React.ReactNode>> = {
  cloudflare: CloudflareIcon,
  vercel: VercelIcon,
};

export function ProviderIcon({ provider, className }: { provider: string; className?: string }) {
  const Icon = providerIcons[provider.toLowerCase().replace(/[.\s]+/g, "")];
  return Icon ? <Icon className={className} /> : null;
}
