// Tremor Button [v0.2.0]

import React from "react";
import { Slot } from "@radix-ui/react-slot";
import { RiLoader3Fill } from "@remixicon/react";
import { tv, type VariantProps } from "tailwind-variants";

import { cx, focusRing } from "../../lib/utils";

const buttonVariants = tv({
  base: [
    // base
    "relative inline-flex items-center justify-center whitespace-nowrap rounded-md border px-2 py-1.5 text-center text-sm font-normal transition-all duration-100 ease-in-out",
    // disabled
    "disabled:pointer-events-none disabled:shadow-none",
    // focus
    focusRing,
  ],
  variants: {
    variant: {
      primary: [
        "font-medium",
        // border
        "border-transparent",
        // text color
        "text-ctexti dark:text-dtext",
        // background color
        "bg-cprimary dark:bg-dprimary",
        // hover color
        "hover:bg-cprimarya dark:hover:bg-dprimarya",
        // disabled
        "disabled:opacity-70 disabled:dark:opacity-70",
      ],
      secondary: [
        // border
        "border-cb dark:border-db",
        // text color
        "text-ctext dark:text-dtext",
        // background color
        "bg-cbg dark:bg-dbg",
        //hover color
        "hover:bg-cbgl dark:hover:bg-dbgl",
        // disabled
        "disabled:text-gray-400",
        "disabled:dark:text-gray-600",
      ],
      light: [
        // base
        "shadow-none",
        // border
        "border-cb dark:border-dbga",
        // text color
        "text-ctext dark:text-dtext",
        // background color
        "bg-cbg dark:bg-db",
        //hover color
        "hover:bg-cbga dark:hover:bg-dprimarya",
        // disabled
        "disabled:bg-gray-100 disabled:text-gray-400",
        "disabled:dark:bg-gray-800 disabled:dark:text-gray-600",
      ],
      ghost: [
        // base
        "shadow-none",
        // border
        "border-transparent",
        // text color
        "text-gray-900 dark:text-gray-50",
        // hover color
        "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800/80",
        // disabled
        "disabled:text-gray-400",
        "disabled:dark:text-gray-600",
      ],
      destructive: [
        // text color
        "text-white",
        // border
        "border-transparent",
        // background color
        "bg-cerr dark:bg-derr opacity-90 ",
        // hover color
        "hover:opacity-100",
        // disabled
        "disabled:bg-red-300 disabled:text-white",
        "disabled:dark:bg-red-950 disabled:dark:text-red-400",
      ],
    },
  },
  defaultVariants: {
    variant: "primary",
  },
});

interface ButtonProps
  extends React.ComponentPropsWithoutRef<"button">,
  VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  isLoading?: boolean;
  loadingText?: string;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      asChild,
      isLoading = false,
      loadingText,
      className,
      disabled,
      variant,
      children,
      ...props
    }: ButtonProps,
    forwardedRef,
  ) => {
    const Component = asChild ? Slot : "button";
    return (
      <Component
        ref={forwardedRef}
        className={cx(buttonVariants({ variant }), className)}
        disabled={disabled || isLoading}
        tremor-id="tremor-raw"
        {...props}
      >
        {isLoading ? (
          <span className="pointer-events-none flex shrink-0 items-center justify-center gap-1.5">
            <RiLoader3Fill
              className="size-4 -mt-0.5 -ml-1 -mr-1 shrink-0 animate-spin absolute"
              aria-hidden="true"
            />
            <span className={cx({ "opacity-0": isLoading })}>
              {loadingText ? loadingText : children}
            </span>
          </span>
        ) : (
          children
        )}
      </Component>
    );
  },
);

Button.displayName = "Button";

export { Button };
