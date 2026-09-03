// Tremor Accordion [v0.0.1]

import React from "react";
import * as AccordionPrimitives from "@radix-ui/react-accordion";
import { RiArrowDownSLine } from "@remixicon/react";

import { cx, focusRing } from "../../lib/utils";

const Accordion = AccordionPrimitives.Root;
Accordion.displayName = "Accordion";

const AccordionItem = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitives.Item>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitives.Item>
>(({ className, ...props }, forwardedRef) => (
  <AccordionPrimitives.Item
    ref={forwardedRef}
    className={cx(
      // base
      "overflow-hidden",
      className,
    )}
    {...props}
  />
));
AccordionItem.displayName = "AccordionItem";

const AccordionTrigger = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitives.Trigger>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitives.Trigger>
>(({ className, children, ...props }, forwardedRef) => (
  <AccordionPrimitives.Header className="flex">
    <AccordionPrimitives.Trigger
      ref={forwardedRef}
      className={cx(
        // base
        "group flex flex-1 items-center justify-between py-2 text-left font-medium transition-all",
        // text color
        "text-ctext dark:text-dtext",
        // disabled
        "data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50",
        focusRing,
        className,
      )}
      {...props}
    >
      {children}
      <RiArrowDownSLine
        className="size-4 shrink-0 text-ctext2 dark:text-dtext2 transition-transform duration-200 group-data-[state=open]:rotate-180"
        aria-hidden="true"
      />
    </AccordionPrimitives.Trigger>
  </AccordionPrimitives.Header>
));
AccordionTrigger.displayName = "AccordionTrigger";

const AccordionContent = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitives.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitives.Content>
>(({ className, children, ...props }, forwardedRef) => (
  <AccordionPrimitives.Content
    ref={forwardedRef}
    className={cx(
      "transform-gpu overflow-hidden text-sm transition-all data-[state=closed]:animate-accordionClose data-[state=open]:animate-accordionOpen",
      className,
    )}
    {...props}
  >
    <div>{children}</div>
  </AccordionPrimitives.Content>
));
AccordionContent.displayName = "AccordionContent";

export { Accordion, AccordionContent, AccordionItem, AccordionTrigger };
