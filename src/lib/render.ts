import { Column } from "./dashboard";

export const formatValue = (value: string | number | boolean, columnType: Column['type']) => {
  if (typeof value === "number") {
    return value;
  }
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  const d = new Date(value)
  if (columnType === "year") {
    return d.getFullYear().toString();
  }
  if (columnType === "month") {
    return d.getFullYear().toString() + '-' + (d.getMonth() + 1).toString().padStart(2, '0')
  }
  if (columnType === "date") {
    return d.getFullYear().toString() + '-' + (d.getMonth() + 1).toString().padStart(2, '0') + '-' + d.getDate().toString().padStart(2, '0')
  }
  if (columnType === "hour") {
    return d.getFullYear().toString() + '-' + (d.getMonth() + 1).toString().padStart(2, '0') + '-' + d.getDate().toString().padStart(2, '0') + ' ' + d.getHours().toString().padStart(2, '0') + ':00'
  }
  if (columnType === "timestamp") {
    return d.getFullYear().toString() + '-' + (d.getMonth() + 1).toString().padStart(2, '0') + '-' + d.getDate().toString().padStart(2, '0') + ' ' + d.getHours().toString().padStart(2, '0') + ':' + d.getMinutes().toString().padStart(2, '0') + ':' + d.getSeconds().toString().padStart(2, '0')
  }
  return value;
};

