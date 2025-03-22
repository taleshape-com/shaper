import { Column } from "./dashboard";

// We interpret the dates as local time to disaply them the same way no matter which timezone a user is in.
// Two people in the same company should be looking at the same timestamps, no matter where they are right now.
// The backend returns dates as unix timestamp integers in milliseconds, but this function works for strings as well
function parseLocalDate(d: string | number) {
  return new Date(d);
}

export const formatValue = (value: string | number | boolean | null | undefined, columnType: Column['type'], shouldFormatNumbers?: boolean) => {
  if (value === null || value === undefined) {
    return ""
  }
  if (columnType === "array" || columnType === "object") {
    return JSON.stringify(value);
  }
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  const d = parseLocalDate(value)
  if (columnType === "year") {
    return d.getFullYear().toString();
  }
  if (columnType === "month") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'short', timeZone: 'UTC' });
  }
  if (columnType === "date") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'numeric', day: 'numeric', timeZone: 'UTC', weekday: 'short' })
  }
  if (columnType === "hour") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC', weekday: 'short' })
  }
  if (columnType === "timestamp") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h24', timeZone: 'UTC' });
  }
  if (columnType === "duration" && !value) {
    return "0"
  }
  if (typeof value === "number") {
    if (shouldFormatNumbers && columnType === "number") {
      return value.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    }
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
    if (columnType === "time") {
      const hours = Math.floor(value / 3600000);
      const minutes = Math.floor((value % 3600000) / 60000);
      const seconds = Math.floor((value % 60000) / 1000);
      const ms = value % 1000;
      const timeString = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
      return ms > 0 ? `${timeString}.${String(ms).padStart(3, '0')}` : timeString;
    }
    if (columnType === "percent") {
      return `${(value * 100).toFixed(2)}%`;
    }
  }
  return value;
};

export const formatCellValue = (value: string | number | boolean | null | undefined) => {
  if (value === null || value === undefined) {
    return ""
  }
  if (typeof value === "boolean") {
    return value ? 0 : 1;
  }
  return value;
}

// Without adding some margins, some bars can end up directly on the edge of the chart
export const getIndexAxisDomain = (minTimeValue: number, maxTimeValue: number) => {
  const margin = (maxTimeValue - minTimeValue) * 0.04
  return [minTimeValue - margin, maxTimeValue + margin]
}

export const isJSONType = (columnType: Column['type']) => {
  return columnType === "array" || columnType === "object"
}
