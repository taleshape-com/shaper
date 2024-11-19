import { Column } from "./dashboard";

// We interpret the dates as local time to disaply them the same way no matter which timezone a user is in.
// Two people in the same company should be looking at the same timestamps, no matter where they are right now.
function parseLocalDate(d: string | number) {
  const date = new Date(d);
  return new Date(date.getTime());
}

export const formatValue = (value: string | number | boolean, columnType: Column['type']) => {
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  const d = parseLocalDate(value)
  if (columnType === "year") {
    return d.getFullYear().toString();
  }
  if (columnType === "month") {
    return d.toLocaleString(navigator.languages, { month: 'short' });
  }
  if (columnType === "date") {
    return d.toLocaleDateString();
  }
  if (columnType === "hour") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }
  if (columnType === "timestamp") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h24' });
  }
  if (typeof value === "number") {
    // duration comes in ms
    if (columnType === "duration") {
      const day = Math.floor(value / 86400000);
      const hours = Math.floor((value % 86400000) / 3600000);
      const minutes = Math.floor((value % 3600000) / 60000);
      const seconds = Math.floor((value % 60000) / 1000);
      const ms = value % 1000;
      const mainParts = []
      if (day > 0) {
        mainParts.push(`${day}d`)
      }
      if (hours > 0) {
        mainParts.push(`${hours}h`)
      }
      if (minutes > 0) {
        mainParts.push(`${minutes}min`)
      }
      if (ms > 0) {
        mainParts.push(`${seconds}.${ms.toString().padStart(3, '0')}s`)
      } else if (seconds > 0) {
        mainParts.push(`${seconds}s`)
      }
      if (mainParts.length === 0) {
        return "0s"
      }
      return mainParts.join(" ")
    }
    return value;
  }
  return value;
};

