// SPDX-License-Identifier: MPL-2.0

import React from "react";
import { RiFullscreenLine, RiFullscreenExitLine } from "@remixicon/react";
import { cx } from "../../lib/utils";
import { translate } from "../../lib/translate";

interface FullscreenButtonProps {
  isFullscreen: boolean;
  onToggle: () => void;
  className?: string;
  id?: string;
}

export const FullscreenButton: React.FC<FullscreenButtonProps> = ({
  isFullscreen,
  onToggle,
  className,
  id,
}) => {
  return (
    <div className="absolute inset-0 pointer-events-none print:hidden">
      <button
        className={cx(
          "absolute top-2 z-50",
          "p-1.5 rounded-md",
          "bg-cbg dark:bg-dbg",
          "border border-cb dark:border-db",
          "text-ctext dark:text-dtext",
          "hover:bg-cbgs dark:hover:bg-dbgs",
          "transition-all duration-100",
          "opacity-0 group-hover:opacity-100",
          "focus:outline-none focus:ring-2 focus:ring-cprimary dark:focus:ring-dprimary",
          "pointer-events-auto",
          isFullscreen ? "opacity-100" : "opacity-0 group-hover:opacity-100",
          className,
        )}
        onClick={(e) => {
          e.stopPropagation();
          onToggle();
        }}
        title={isFullscreen ? translate("Exit fullscreen") : translate("Fullscreen")}
        id={id}
      >
        {isFullscreen ? (
          <RiFullscreenExitLine className="size-4" />
        ) : (
          <RiFullscreenLine className="size-4" />
        )}
      </button>
    </div>
  );
};
