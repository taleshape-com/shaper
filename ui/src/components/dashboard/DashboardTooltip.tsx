// SPDX-License-Identifier: MPL-2.0

import React, { useState, useRef, useEffect, useCallback } from "react";
import * as PopoverPrimitives from "@radix-ui/react-popover";
import { RiInformationLine } from "@remixicon/react";
import { cx } from "../../lib/utils";
import TextWithLinks from "../TextWithLinks";

export interface DashboardTooltipProps {
  content?: string | null;
  size?: "sm" | "md";
  className?: string;
  side?: "top" | "bottom" | "left" | "right";
  align?: "start" | "center" | "end";
  ariaLabel?: string;
}

export const DashboardTooltip: React.FC<DashboardTooltipProps> = ({
  content,
  size = "sm",
  className,
  side = "top",
  align = "center",
  ariaLabel = "More information",
}) => {
  const [open, setOpen] = useState(false);
  const [isPinned, setIsPinned] = useState(false);
  const openTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimers = useCallback(() => {
    if (openTimerRef.current) {
      clearTimeout(openTimerRef.current);
      openTimerRef.current = null;
    }
    if (closeTimerRef.current) {
      clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      clearTimers();
    };
  }, [clearTimers]);

  if (!content) {
    return null;
  }

  const handleMouseEnter = () => {
    if (isPinned) return;
    if (closeTimerRef.current) {
      clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    if (!open) {
      openTimerRef.current = setTimeout(() => {
        setOpen(true);
      }, 200);
    }
  };

  const handleMouseLeave = () => {
    if (isPinned) return;
    if (openTimerRef.current) {
      clearTimeout(openTimerRef.current);
      openTimerRef.current = null;
    }
    if (open) {
      closeTimerRef.current = setTimeout(() => {
        setOpen(false);
      }, 150);
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    clearTimers();
    if (isPinned) {
      setIsPinned(false);
      setOpen(false);
    } else {
      setIsPinned(true);
      setOpen(true);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape" && open) {
      e.preventDefault();
      e.stopPropagation();
      clearTimers();
      setIsPinned(false);
      setOpen(false);
    }
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      clearTimers();
      setIsPinned(false);
      setOpen(false);
    } else {
      setOpen(true);
    }
  };

  let container = typeof document !== "undefined" ? document.querySelector(".shaper-scope") as HTMLElement | null : null;
  if (container === (typeof document !== "undefined" ? document.querySelector("html") : null)) {
    container = document.body;
  }

  return (
    <PopoverPrimitives.Root open={open} onOpenChange={handleOpenChange}>
      <PopoverPrimitives.Trigger asChild>
        <button
          type="button"
          aria-label={ariaLabel}
          aria-expanded={open}
          onClick={handleClick}
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
          onKeyDown={handleKeyDown}
          className={cx(
            "shaper-tooltip-anchor shaper-tooltip-trigger inline-flex items-center justify-center shrink-0 cursor-pointer rounded text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors",
            "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cring focus-visible:dark:ring-dring",
            "print:hidden",
            className,
          )}
        >
          <RiInformationLine
            className={cx("shaper-tooltip-anchor", size === "md" ? "size-5" : "size-4")}
            aria-hidden="true"
          />
        </button>
      </PopoverPrimitives.Trigger>
      <PopoverPrimitives.Portal container={container ?? undefined}>
        <PopoverPrimitives.Content
          side={side}
          align={align}
          sideOffset={6}
          avoidCollisions
          onOpenAutoFocus={(e) => {
            if (!isPinned) {
              e.preventDefault();
            }
          }}
          onCloseAutoFocus={(e) => {
            if (!isPinned) {
              e.preventDefault();
            }
          }}
          onMouseEnter={() => {
            if (!isPinned && closeTimerRef.current) {
              clearTimeout(closeTimerRef.current);
              closeTimerRef.current = null;
            }
          }}
          onMouseLeave={() => {
            if (!isPinned && open) {
              closeTimerRef.current = setTimeout(() => {
                setOpen(false);
              }, 150);
            }
          }}
          className={cx(
            "shaper-tooltip shaper-tooltip-content z-50 max-w-xs sm:max-w-sm rounded-md px-3 py-2 text-xs leading-relaxed shadow-md",
            "text-ctexti dark:text-dtexti bg-cbgi dark:bg-dbgi",
            "focus:outline-none select-text",
            "will-change-[transform,opacity]",
            "data-[side=bottom]:animate-slideDownAndFade data-[side=left]:animate-slideLeftAndFade data-[side=right]:animate-slideRightAndFade data-[side=top]:animate-slideUpAndFade data-[state=closed]:animate-hide",
          )}
        >
          <TextWithLinks text={content} />
          <PopoverPrimitives.Arrow
            className="border-none fill-cbgi dark:fill-dbgi"
            width={12}
            height={7}
            aria-hidden="true"
          />
        </PopoverPrimitives.Content>
      </PopoverPrimitives.Portal>
    </PopoverPrimitives.Root>
  );
};

export default DashboardTooltip;
