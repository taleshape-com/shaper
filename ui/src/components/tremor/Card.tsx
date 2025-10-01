// Tremor Card [v0.0.2]

import React from "react";
import { Slot } from "@radix-ui/react-slot";

import { cx } from "../../lib/utils";

interface CardProps extends React.ComponentPropsWithoutRef<"div"> {
  asChild?: boolean
}

const Card = React.forwardRef<HTMLDivElement, CardProps>(
  ({ className, asChild, ...props }, forwardedRef) => {
    const Component = asChild ? Slot : "div";
    return (
      <Component
        ref={forwardedRef}
        className={cx(
          // base
          "relative rounded-sm border",
          // background color
          "bg-cbg dark:bg-dbg",
          // border color
          "border-cb dark:border-db",
          className,
        )}
        tremor-id="tremor-raw"
        {...props}
      />
    );
  },
);

Card.displayName = "Card";

export { Card, type CardProps };
