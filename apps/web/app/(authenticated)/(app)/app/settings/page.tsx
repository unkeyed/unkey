"use client"

import * as React from "react"
import Link from "next/link"

import { cn } from "@/lib/utils"
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  NavigationMenuTrigger,
  navigationMenuTriggerStyle,
} from "@/components/ui/navigation-menu"


export default function Page() {

    const navigation: {label: string, href: string,active?:boolean}[] = [
        {
            label: "General",
            href: "/app/settings",
        },
        {
            label: "Root Keys",
            href: "/app/settings/root-keys",
        },
        {
            label: "Billing",
            href: "/app/stripe",
        },
        {
            label: "Usage",
            href: "/app/settings/usage",
        }
    ]

  return (
    <div>

    <header className="">

    <NavigationMenu>
      <NavigationMenuList>
        {navigation.map(({ label, href, active }) => (

            <NavigationMenuItem>
          <Link href={href} legacyBehavior passHref>
            <NavigationMenuLink className={navigationMenuTriggerStyle()}>
              {label}
            </NavigationMenuLink>
          </Link>
        </NavigationMenuItem>
            ))}
      </NavigationMenuList>
    </NavigationMenu>
    </header>
    <hr/>
    <main>
        x
    </main>
    </div>

  )
}
