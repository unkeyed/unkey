import React from "react"
import Link from "next/link"
import { Button } from "@unkey/ui"
import { ProgressCircle } from "@/app/(app)/settings/billing/components/usage"
import { Particles } from "@/components/ui/particles"
import { createContext } from "@/lib/create-context"
import { cn } from "@/lib/utils"

/* ----------------------------------------------------------------------------
 * UsageInsight - Root
 * --------------------------------------------------------------------------*/

type DivElement = React.ElementRef<"div">
type DivProps = React.ComponentPropsWithoutRef<"div">
type UsageContextValue = {
  plan: string
  current: number
  max: number
}
export interface UsageRootProps extends DivProps {
  plan: string
  current: number
  max: number
}

const ROOT_NAME = "UsageRoot"

const [UsageProvider, useUsageContext] =
  createContext<UsageContextValue>(ROOT_NAME)

export const Root = React.forwardRef<DivElement, UsageRootProps>(
  (props, ref) => {
    const { plan, current, max, className, children, ...rootProps } = props
    const { format } = Intl.NumberFormat(undefined, { notation: "compact" })

    return (
      <UsageProvider plan={plan} current={current} max={max}>
        <div
          {...rootProps}
          ref={ref}
          className={cn(
            "relative flex flex-col bg-background border border-border rounded-xl pt-2.5 pb-2 px-2 group",
            { className }
          )}
        >
          <Particles
            className="absolute inset-0 duration-500 opacity-0 pointer-events-none group-hover:opacity-75"
            quantity={50}
            color={plan === "Free" ? "#818181" : "#d6b300"}
            vx={0.1}
            vy={-0.1}
          />

          <div className="z-10">
            <div className="flex items-center justify-between px-2">
              <div className="flex items-center justify-start gap-2">
                <h2 className="text-gray-12 text-lg capitalize">{plan}</h2>
                <span
                  className={cn(
                    "h-6 rounded-md px-2 text-gray-11 text-xs bg-secondary flex items-center justify-center",
                    {
                      "bg-warning-11": current / max >= 0.94,
                    }
                  )}
                >
                  {format((current / max) * 100)}%
                </span>
              </div>

              {current / max >= 0.94 && (
                <Button variant="primary" size="sm">
                  <Link href="/settings/billing">Upgrade</Link>
                </Button>
              )}
            </div>

            {children}
          </div>
        </div>
      </UsageProvider>
    )
  }
)

Root.displayName = ROOT_NAME

/* ----------------------------------------------------------------------------
 * UsageInsight - Details
 * --------------------------------------------------------------------------*/

const DETAILS_NAME = "UsageDetails"

export const Details = React.forwardRef<DivElement, DivProps>((props, ref) => {
  const { className, children, ...detailsProps } = props

  return (
    <div
      {...detailsProps}
      ref={ref}
      className={cn(
        "h-0 group-hover:h-12 duration-500 ease-out w-full overflow-hidden group-hover:opacity-100 opacity-0 flex flex-col gap-4 group-hover:mt-4 mt-1.5",
        { className }
      )}
    >
      {children}
    </div>
  )
})

Details.displayName = DETAILS_NAME

/* ----------------------------------------------------------------------------
 * UsageInsight - Item
 * --------------------------------------------------------------------------*/

export interface UsageItemProps extends DivProps {
  item?: {
    current: number
    max: number
  }
  title: string
  description?: string
  color?: string
}

const ITEM_NAME = "UsageItem"

export const Item = React.forwardRef<DivElement, UsageItemProps>(
  (props, ref) => {
    const { item, title, description, color, className, ...itemProps } = props
    const { format } = Intl.NumberFormat(undefined, { notation: "compact" })
    const { current, max } = useUsageContext("UsageItem")

    return (
      <div
        {...itemProps}
        ref={ref}
        className={cn(
          "flex items-start justify-start px-2 gap-3 group-hover:opacity-100 opacity-0 transition-opacity duration-500 ease-out delay-150",
          { className }
        )}
      >
        <ProgressCircle
          value={item?.current ?? current}
          max={item?.max ?? max}
          color={color ?? "#f76e19"}
        />
        <div className="flex flex-col gap-2 justify-start items-start select-none">
          <h6 className="text-gray-12 text-sm leading-none">{title}</h6>
          <p className="text-xs font-normal line-clamp-1">
            {format(item?.current ?? current)} of {format(item?.max ?? max)}{" "}
            {description}
          </p>
        </div>
      </div>
    )
  }
)

Item.displayName = ITEM_NAME

/* ----------------------------------------------------------------------------
 * UsageInsight - Footer
 * --------------------------------------------------------------------------*/

const FOOTER_NAME = "UsageFooter"

export const Footer = React.forwardRef<DivElement, DivProps>((props, ref) => {
  const { className, children, ...detailsProps } = props

  return (
    <div {...detailsProps} ref={ref} className={cn({ className })}>
      {children}
    </div>
  )
})

Footer.displayName = FOOTER_NAME

/* -------------------------------------------------------------------------------------------------
 * Exports
 * -----------------------------------------------------------------------------------------------*/

export const UsageInsight = Object.assign(
  {},
  {
    Root,
    Details,
    Item,
    Footer,
  }
)
