// SPDX-License-Identifier: MPL-2.0

const MONTHS_SHORT = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
];

/**
 * Adds the specified number of years to a given date.
 */
export function addYears (date: Date, amount: number): Date {
  const newDate = new Date(date.getTime());
  newDate.setFullYear(newDate.getFullYear() + amount);
  return newDate;
}

/**
 * Checks if two dates are in the same month of the same year.
 */
export function isSameMonth (dateLeft: Date, dateRight: Date): boolean {
  return (
    dateLeft.getFullYear() === dateRight.getFullYear() &&
    dateLeft.getMonth() === dateRight.getMonth()
  );
}

/**
 * Formats a Date object into a string. Supports "MMM dd, yyyy" and "MMM, yyyy".
 */
export function formatDate (date: Date, formatStr: string): string {
  const day = String(date.getDate()).padStart(2, "0");
  const month = MONTHS_SHORT[date.getMonth()];
  const year = date.getFullYear();

  if (formatStr === "MMM dd, yyyy") {
    return `${month} ${day}, ${year}`;
  }
  if (formatStr === "MMM, yyyy") {
    return `${month}, ${year}`;
  }

  // Fallback token replacement if needed
  return formatStr
    .replace("MMM", month)
    .replace("dd", day)
    .replace("yyyy", String(year));
}
