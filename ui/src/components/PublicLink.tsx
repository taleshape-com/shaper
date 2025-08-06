// SPDX-License-Identifier: MPL-2.0

import { RiExternalLinkLine } from "@remixicon/react";
import { translate } from "../lib/translate";

export function PublicLink({ href }: { href: string }) {
  return (
    <a
      href={href}
      target="_blank"
      className="py-2 px-2 text-sm text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext underline transition-colors duration-200 inline-block">
      {translate("Public Link")}
      <RiExternalLinkLine className="size-3.5 inline ml-1 -mt-1 fill-ctext2 dark:fill-dtext2" />
    </a>
  );
}

