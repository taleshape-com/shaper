import React, { useContext } from "react";
import { MenuContext } from "../contexts/MenuContext";
import { cx } from "../lib/utils";

export function DashboardWrapper ({ children, className }: {
  children: React.ReactNode;
  className?: string;
}) {
  const { isMenuOpen } = useContext(MenuContext);
  console.log({ isMenuOpen });
  return <div className={cx(className, {
    "xl:px-[4%] 2xl:px-[6%] 4xl:px-[10%]": isMenuOpen,
    // same as above minus 224px (the width of the menu)
    "min-[1056px]:px-[4%] min-[1312px]:px-[6%] min-[1724px]:px-[10%]": !isMenuOpen,
  })}>
    {children}
  </div>;
}
