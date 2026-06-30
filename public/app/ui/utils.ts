import type { Tone } from "./types";

export function formatDateTime(value: string | number | null | undefined) {
  if (!value) {
    return "Chưa thiết lập";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Thời gian không hợp lệ";
  }

  return date.toLocaleString("vi-VN", { hour12: false });
}

export function formatHours(hours: number) {
  if (hours < 0) {
    return "Không khả dụng";
  }

  if (hours < 1) {
    return `${Math.round(hours * 60)} phút`;
  }

  return `${hours.toFixed(hours < 10 ? 1 : 0)} giờ`;
}

export function getStatusTone(status: string): Tone {
  if (status === "valid") {
    return "success";
  }
  if (status === "expiringSoon") {
    return "warning";
  }
  if (status === "expired" || status === "invalid") {
    return "danger";
  }
  return "default";
}
