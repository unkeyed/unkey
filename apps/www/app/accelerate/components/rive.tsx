"use client";

import {
  Event,
  EventType,
  RiveEventType,
  useRive,
  useStateMachineInput,
} from "@rive-app/react-canvas-lite";
import React from "react";

const stateMachines = "sms";

export const RiveAccelerate = ({ day }: { day: number }) => {
  const [done, setDone] = React.useState(false);

  const r = useRive({
    src: "assets/accelerate/rive/accelerate.riv",
    stateMachines,
    autoplay: true,
  });

  function onClickDay(day: number) {
    let el = document.getElementById(`anchor_d${day}`);
    if (!el) {
      el = document.getElementById("anchor_d1");
    }
    if (!el) {
      console.error("Could not find anchor element");
      return;
    }
    el.scrollIntoView();
  }

  const highlight = useStateMachineInput(r.rive, stateMachines, "highlight");

  // Wait until the rive object is instantiated before adding the Rive
  // event listener
  React.useEffect(() => {
    if (done || !r.rive) {
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

    if (day > 0 && highlight) {
      highlight.value = day + 1;
    }

    r.rive.on(EventType.RiveEvent, onRiveEventReceived);

    setDone(true);
  }, [done, r.rive]);

  return <r.RiveComponent />;
};
