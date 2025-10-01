// Tremor Dialog [v0.0.1]

import React from "react";
import * as DialogPrimitives from "@radix-ui/react-dialog";

import { cx, focusRing } from "../../lib/utils";

const Dialog = (
  props: React.ComponentPropsWithoutRef<typeof DialogPrimitives.Root>,
) => {
  return <DialogPrimitives.Root {...props} />;
};
Dialog.displayName = "Dialog";

const DialogTrigger = DialogPrimitives.Trigger;

DialogTrigger.displayName = "DialogTrigger";

const DialogClose = DialogPrimitives.Close;

DialogClose.displayName = "DialogClose";

const DialogPortal = DialogPrimitives.Portal;

DialogPortal.displayName = "DialogPortal";

const DialogOverlay = React.forwardRef<
  React.ElementRef<typeof DialogPrimitives.Overlay>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitives.Overlay>
>(({ className, ...props }, forwardedRef) => {
  return (
    <DialogPrimitives.Overlay
      ref={forwardedRef}
      className={cx(
        // base
        "fixed inset-0 z-50 overflow-y-auto",
        // background color
        "bg-black/55",
        // transition
        "data-[state=open]:animate-dialogOverlayShow",
        className,
      )}
      {...props}
    />
  );
});

DialogOverlay.displayName = "DialogOverlay";

const DialogContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitives.Content>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitives.Content>
>(({ className, ...props }, forwardedRef) => {
  return (
    <DialogPortal container={document.querySelector(".shaper-scope") === document.querySelector("html") ? document.body : document.querySelector(".shaper-scope")}>
      <DialogOverlay className="absolute">
        <DialogPrimitives.Content
          ref={forwardedRef}
          className={cx(
            // base
            "absolute left-1/2 top-1/2 z-50 w-[95vw] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-md p-6 space-y-4",
            // background color
            "bg-cbg dark:bg-dbg",
            // transition
            "data-[state=open]:animate-dialogContentShow",
            focusRing,
            className,
          )}
          tremor-id="tremor-raw"
          {...props}
        />
      </DialogOverlay>
    </DialogPortal>
  );
});

DialogContent.displayName = "DialogContent";

const DialogHeader = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => {
  return <div className={cx("flex flex-col gap-y-1", className)} {...props} />;
};

DialogHeader.displayName = "DialogHeader";

const DialogTitle = React.forwardRef<
  React.ElementRef<typeof DialogPrimitives.Title>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitives.Title>
>(({ className, ...props }, forwardedRef) => (
  <DialogPrimitives.Title
    ref={forwardedRef}
    className={cx(
      // base
      "text-lg font-semibold",
      // text color
      "text-ctext dark:text-dtext",
      className,
    )}
    {...props}
  />
));

DialogTitle.displayName = "DialogTitle";

const DialogDescription = React.forwardRef<
  React.ElementRef<typeof DialogPrimitives.Description>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitives.Description>
>(({ className, ...props }, forwardedRef) => {
  return (
    <DialogPrimitives.Description
      ref={forwardedRef}
      className={cx("text-ctext2 dark:text-ctext2", className)}
      {...props}
    />
  );
});

DialogDescription.displayName = "DialogDescription";

const DialogFooter = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => {
  return (
    <div
      className={cx(
        "flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2",
        className,
      )}
      {...props}
    />
  );
};

DialogFooter.displayName = "DialogFooter";

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
};
