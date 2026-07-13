const LOCAL_DATE_PATTERN =
  /^(\d{4})-(\d{2})-(\d{2})(?:[T\s](\d{2}):(\d{2})(?::(\d{2}))?)?$/;

function isValidDate(date: Date): boolean {
  return !Number.isNaN(date.getTime());
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

export function parseDatePickerValue(
  value: Date | string | undefined
): Date | undefined {
  if (value instanceof Date) {
    return isValidDate(value) ? new Date(value) : undefined;
  }
  if (!value) {
    return undefined;
  }

  const localMatch = LOCAL_DATE_PATTERN.exec(value);
  if (localMatch) {
    const [, year, month, day, hour = '0', minute = '0', second = '0'] =
      localMatch;
    const date = new Date(
      Number(year),
      Number(month) - 1,
      Number(day),
      Number(hour),
      Number(minute),
      Number(second)
    );
    const matchesInput =
      date.getFullYear() === Number(year) &&
      date.getMonth() === Number(month) - 1 &&
      date.getDate() === Number(day) &&
      date.getHours() === Number(hour) &&
      date.getMinutes() === Number(minute) &&
      date.getSeconds() === Number(second);

    return matchesInput ? date : undefined;
  }

  const parsed = new Date(value);
  return isValidDate(parsed) ? parsed : undefined;
}

export function serializeDatePickerValue(
  date: Date | undefined,
  includeTime: boolean
): string {
  if (!date || !isValidDate(date)) {
    return '';
  }

  const datePart = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(
    date.getDate()
  )}`;
  if (!includeTime) {
    return datePart;
  }

  return `${datePart}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(
    date.getSeconds()
  )}`;
}
