//@ts-nocheck 

import { Border } from '@/components/landingComponents/Border'
import { FadeIn, FadeInStagger } from '@/components/landingComponents/FadeIn'

export function StatList({ children, ...props } : {
  children?: React.ReactNode | React.ReactNode[],
  props : any}) {
  return (
    <FadeInStagger {...props}>
      <dl className="grid grid-cols-1 gap-10 sm:grid-cols-2 lg:auto-cols-fr lg:grid-flow-col lg:grid-cols-none">
        {children}
      </dl>
    </FadeInStagger>
  )
}

export function StatListItem({ label, value } : {
  label?: string,
  value?: string,
}) {
  return (
    <Border as={FadeIn} position="left" className="flex flex-col-reverse pl-8">
      <dt className="mt-2 text-base text-neutral-600">{label}</dt>
      <dd className="font-display text-3xl font-semibold text-neutral-950 sm:text-4xl">
        {value}
      </dd>
    </Border>
  )
}
