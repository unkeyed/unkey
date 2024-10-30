"use client";

import {
  EventType,
  RiveEventType,
  useRive,
  useStateMachineInput,
} from "@rive-app/react-canvas-lite";
import React from "react";

type RiveAccelerateProps = {
  day: number;
};

export const RiveAccelerate = (props: RiveAccelerateProps) => {
  const [isReady, setIsReady] = React.useState(false);
  const [isDesktop, setIsDesktop] = React.useState(false);
  const stateMachines = isDesktop ? "sms" : "sms_mobile";

  React.useEffect(() => {
    const media = window.matchMedia("(min-width: 768px)");
    if (media.matches !== isDesktop) {
      setIsDesktop(media.matches);
    }
    const listener = () => setIsDesktop(media.matches);
    window.addEventListener("resize", listener);
    setIsReady(true);
    return () => window.removeEventListener("resize", listener);
  }, [isDesktop]);

  if (!isReady) {
    return <></>;
  }

  return <RiveAccelerateAsset day={props.day} stateMachines={stateMachines} />;
};

const RiveAccelerateAsset = ({ day, stateMachines }: { day: number; stateMachines: string }) => {
  const [done, setDone] = React.useState(false);

  const r = useRive({
    src: "assets/accelerate/rive/accelerate.riv",
    stateMachines,
    autoplay: true,
  });

  const smVarHighlight = useStateMachineInput(r.rive, stateMachines, "highlight");
  const smVarUnlockedUntil = useStateMachineInput(r.rive, stateMachines, "unlocked_until");

  function onClickDay(day: number) {
    let el = document.getElementById(`day_${day}`);
    if (!el) {
      el = document.getElementById("day_1");
    }
    if (!el) {
      console.error("Could not find anchor element");
      return;
    }
    el.scrollIntoView();
  }

  // Wait until the rive object is instantiated before adding the Rive
  // event listener
  React.useEffect(() => {
    if (done || !r.rive || smVarHighlight == null || smVarUnlockedUntil == null) {
      return;
    }

    const onRiveEventReceived = (riveEvent: any) => {
      const eventData = riveEvent.data;
      if (eventData.type === RiveEventType.General) {
        if (eventData.name === "click_d1") {
          onClickDay(1);
        }
        if (eventData.name === "click_d2") {
          onClickDay(2);
        }
        if (eventData.name === "click_d3") {
          onClickDay(3);
        }
        if (eventData.name === "click_d4") {
          onClickDay(4);
        }
        if (eventData.name === "click_d5") {
          onClickDay(5);
        }
        if (eventData.name === "click_d6") {
          onClickDay(6);
        }
      }
    };

    if (day > 0 && smVarHighlight !== null && smVarUnlockedUntil !== null) {
      smVarUnlockedUntil.value = day;
      smVarHighlight.value = day;
    }

    r.rive.on(EventType.RiveEvent, onRiveEventReceived);

    setDone(true);
  }, [done, r.rive, smVarHighlight, smVarUnlockedUntil]);

  return <r.RiveComponent />;
};
