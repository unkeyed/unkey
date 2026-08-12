"use client";

import {
  UserProfile,
  UserSecurity,
  UsersManagement,
  WorkOsWidgets,
  type WorkOsWidgetsProps,
} from "@workos-inc/widgets";
import type React from "react";

const UNKEY_WORKOS_THEME = {
  accentColor: "gray",
  appearance: "inherit",
  fontFamily: "var(--font-geist-sans)",
  grayColor: "gray",
  hasBackground: false,
  panelBackground: "solid",
  radius: "medium",
  scaling: "100%",
} satisfies NonNullable<WorkOsWidgetsProps["theme"]>;

const UNKEY_WORKOS_ELEMENTS = {
  avatar: {
    color: "gray",
    highContrast: true,
    radius: "full",
    size: "2",
    variant: "soft",
  },
  badge: {
    color: "gray",
    radius: "medium",
    size: "1",
    variant: "soft",
  },
  /*
   * Sizing lives in styles.css only: setting it here too would tie on
   * specificity and leave the winner to stylesheet import order.
   */
  dialog: {
    align: "center",
    size: "3",
  },
  dropdown: {
    color: "gray",
    highContrast: true,
    /*
     * Without it, selects use Radix's item-aligned mode and open as a trigger-width sheet on top
     * of the trigger; popper opens them below the trigger like every other
     * dashboard select.
     */
    position: "popper",
    size: "2",
    variant: "solid",
  },
  primaryButton: {
    color: "gray",
    highContrast: true,
    radius: "medium",
    size: "2",
    variant: "solid",
  },
  secondaryButton: {
    color: "gray",
    highContrast: true,
    radius: "medium",
    size: "2",
    variant: "surface",
  },
  destructiveButton: {
    color: "red",
    radius: "medium",
    size: "2",
    variant: "solid",
  },
  iconButton: {
    color: "gray",
    highContrast: true,
    radius: "medium",
    size: "2",
    variant: "ghost",
  },
  label: {
    color: "gray",
    highContrast: true,
    size: "2",
    weight: "medium",
  },
  primaryMenuItem: {
    color: "gray",
  },
  destructiveMenuItem: {
    color: "red",
  },
  select: {
    color: "gray",
    radius: "medium",
    variant: "surface",
  },
  textfield: {
    color: "gray",
    radius: "medium",
    size: "2",
    variant: "surface",
  },
} satisfies NonNullable<WorkOsWidgetsProps["elements"]> & {
  dropdown: { position: "popper" };
};

function UnkeyWorkOsWidgets({ children }: { children: React.ReactNode }) {
  return (
    <WorkOsWidgets
      className="unkey-workos-widgets"
      elements={UNKEY_WORKOS_ELEMENTS}
      theme={UNKEY_WORKOS_THEME}
    >
      {children}
    </WorkOsWidgets>
  );
}

/**
 * Renders WorkOS-managed profile and security controls.
 *
 * User Sessions is intentionally excluded because session management is not
 * part of the dashboard account experience.
 */
export function ManagedUserWidgets({
  getAccessToken,
}: {
  getAccessToken: () => Promise<string>;
}) {
  return (
    <UnkeyWorkOsWidgets>
      <div className="flex flex-col gap-8">
        <section aria-labelledby="profile-settings-heading" className="flex flex-col gap-3">
          <h2
            id="profile-settings-heading"
            tabIndex={-1}
            className="m-0 text-lg font-medium outline-none"
          >
            Profile
          </h2>
          <UserProfile authToken={getAccessToken} />
        </section>
        <section aria-labelledby="security-settings-heading" className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <h2 id="security-settings-heading" className="m-0 text-lg font-medium">
              Security
            </h2>
            <p className="m-0 text-sm text-gray-11">
              Enroll in MFA here even when your organization does not require it.
            </p>
          </div>
          <UserSecurity authToken={getAccessToken} />
        </section>
      </div>
    </UnkeyWorkOsWidgets>
  );
}

/**
 * Renders WorkOS-managed organization members, invitations, and roles.
 */
export function ManagedUsersWidget({
  getAccessToken,
}: {
  getAccessToken: () => Promise<string>;
}) {
  return (
    <UnkeyWorkOsWidgets>
      <UsersManagement authToken={getAccessToken} />
    </UnkeyWorkOsWidgets>
  );
}
