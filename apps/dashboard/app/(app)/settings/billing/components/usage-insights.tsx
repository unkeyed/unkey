import { ProgressCircle } from "@/app/(app)/settings/billing/components/usage"
import { createContext } from "@/lib/create-context"
import { cn } from "@/lib/utils"
import type { Workspace } from "@unkey/db"
import { Button } from "@unkey/ui"
import { motion } from "framer-motion"
import Link from "next/link"
import React, { useState, useEffect, useMemo } from "react"

/* ----------------------------------------------------------------------------
 * UsageInsight - Root
 * --------------------------------------------------------------------------*/

type DivElement = React.ElementRef<typeof motion.div>
type DivProps = React.ComponentPropsWithoutRef<typeof motion.div>
type UsageContextValue = {
  plan: string
  current: number
  max: number
}
export interface UsageRootProps extends DivProps {
  plan: string
  current: number
  max: number
  isLoading?: boolean
  children: React.ReactNode | React.ReactNode[]
}

const ROOT_NAME = "UsageRoot"

const [UsageProvider, useUsageContext] =
  createContext<UsageContextValue>(ROOT_NAME)

export const Root = React.forwardRef<DivElement, UsageRootProps>(
  (props, ref) => {
    const {
      plan,
      current,
      max,
      className,
      children,
      isLoading = false,
      ...rootProps
    } = props

    const [hasBeenMounted, setHasBeenMounted] = useState(false)
    const [hasInitialData, setHasInitialData] = useState(false)

    const cachedData = useMemo(() => {
      if (plan && typeof current === "number" && typeof max === "number") {
        return { plan, current, max }
      }
      return null
    }, [plan, current, max])

    useEffect(() => {
      setHasBeenMounted(true)
    }, [])

    useEffect(() => {
      if (!hasInitialData && cachedData) {
        setHasInitialData(true)
      }
    }, [cachedData, hasInitialData])

    const dataToUse = useMemo(() => {
      if (isLoading && cachedData) {
        return cachedData
      }
      if (plan && typeof current === "number" && typeof max === "number") {
        return { plan, current, max }
      }
      return cachedData
    }, [isLoading, cachedData, plan, current, max])

    if (!dataToUse) {
      return null
    }

    const shouldAnimate = !hasBeenMounted && hasInitialData

    return (
      <UsageProvider
        plan={dataToUse.plan}
        current={dataToUse.current}
        max={dataToUse.max}
      >
        {!shouldAnimate && (
          <motion.div
            {...rootProps}
            layout
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "116px" }}
            transition={{
              type: "spring",
              stiffness: 243,
              damping: 34,
              mass: 1,
              delay: 1,
            }}
            ref={ref}
            className={cn(
              "relative flex flex-col bg-background border border-border rounded-xl p-4 group overflow-hidden w-full",
              { className }
            )}
          >
            <div className="z-10">
              <div className="flex items-center justify-between">
                <div className="flex items-center justify-start gap-2">
                  <h2 className="text-gray-12 text-lg capitalize">
                    {dataToUse.plan}
                  </h2>
                </div>

                {dataToUse.current / dataToUse.max >= 0.94 ? (
                  <Button variant="primary" size="sm">
                    <Link href="/settings/billing">Upgrade</Link>
                  </Button>
                ) : (
                  <Button variant="outline" size="sm">
                    <Link href="/settings/billing">Manage Plan</Link>
                  </Button>
                )}
              </div>

              {children}
            </div>
          </motion.div>
        )}
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
    <motion.div
      initial={{ opacity: 0, x: -16 }}
      animate={{ opacity: 1, x: -0 }}
      transition={{
        ease: "easeInOut",
        duration: 0.4,
        opacity: { duration: 0.8, delay: 0.1 },
      }}
      {...detailsProps}
      ref={ref}
      className={cn(
        "h-16 duration-500 ease-out w-full flex flex-col items-start justify-start pt-4",
        {
          className,
        }
      )}
    >
      {children}
    </motion.div>
  )
})

Details.displayName = DETAILS_NAME

/* ----------------------------------------------------------------------------
 * UsageInsight - Item
 * --------------------------------------------------------------------------*/
type PrimitiveDivElement = React.ElementRef<"div">
type PrimitiveDivProps = React.ComponentPropsWithoutRef<"div">

export interface UsageItemProps extends PrimitiveDivProps {
  item?: {
    current: number
    max: number
  }
  title: string
  description?: string
  color?: string
}

const ITEM_NAME = "UsageItem"

export const Item = React.forwardRef<PrimitiveDivElement, UsageItemProps>(
  (props, ref) => {
    const { item, title, description, color, className, ...itemProps } = props
    const { format } = Intl.NumberFormat(undefined, { notation: "compact" })
    const { current, max } = useUsageContext("UsageItem")

    return (
      <div
        {...itemProps}
        ref={ref}
        className={cn("flex items-start justify-start gap-3", {
          className,
        })}
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

/* -------------------------------------------------------------------------------------------------
 * Exports
 * -----------------------------------------------------------------------------------------------*/

export const UsageInsight = Object.assign(
  {},
  {
    Root,
    Details,
    Item,
  }
)
