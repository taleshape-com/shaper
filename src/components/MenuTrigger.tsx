import React, { useEffect, useContext } from "react";
import { RiMenuLine } from "@remixicon/react";
import { MenuContext } from "../contexts/MenuContext";

export function MenuTrigger({
  children,
  className,
}: {
  children?: React.ReactNode;
  className?: string;
}) {
  const { isMenuOpen, setIsMenuOpen, setExtraContent } = useContext(MenuContext);

  useEffect(() => {
    setExtraContent(children);
  }, [children, setExtraContent]);

  if (isMenuOpen) {
    return null;
  }

  return (
    <button
      className={className}
      onClick={() => {
        setIsMenuOpen(true);
      }}
    >
      <RiMenuLine className="py-1 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
    </button>
  );
}

