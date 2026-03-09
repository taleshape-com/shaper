// SPDX-License-Identifier: MPL-2.0

export interface RetryOptions {
  maxRetries?: number;
  baseDelayMs?: number;
  retryableStatuses?: number[];
}

const DEFAULT_MAX_RETRIES = 3;
const DEFAULT_BASE_DELAY_MS = 1000;
const DEFAULT_RETRYABLE_STATUSES = [429, 500, 502, 503, 504];

function isRetryable (status: number, retryableStatuses: number[]) {
  return retryableStatuses.includes(status);
}

function getDelay (attempt: number, baseDelayMs: number) {
  // Exponential backoff with jitter: base * 2^attempt * (0.5..1.0)
  const exponential = baseDelayMs * Math.pow(2, attempt);
  const jitter = 0.5 + Math.random() * 0.5;
  return exponential * jitter;
}

function sleep (ms: number, signal?: AbortSignal) {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(signal.reason ?? new DOMException("Aborted", "AbortError"));
      return;
    }
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(signal.reason ?? new DOMException("Aborted", "AbortError"));
      },
      { once: true },
    );
  });
}

// Wrapper around fetch that retries on failure with exponential backoff
export async function fetchWithRetry (
  input: RequestInfo | URL,
  init?: RequestInit,
  retryOptions?: RetryOptions,
) {
  const maxRetries = retryOptions?.maxRetries ?? DEFAULT_MAX_RETRIES;
  const baseDelayMs = retryOptions?.baseDelayMs ?? DEFAULT_BASE_DELAY_MS;
  const retryableStatuses = retryOptions?.retryableStatuses ?? DEFAULT_RETRYABLE_STATUSES;

  const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;

  for (let attempt = 0; ; attempt++) {
    try {
      const response = await fetch(input, init);

      if (attempt < maxRetries && isRetryable(response.status, retryableStatuses)) {
        console.warn(
          `Attempt ${attempt + 1}/${maxRetries + 1} failed with status ${response.status} for ${url}. Retrying...`,
        );
        await sleep(getDelay(attempt, baseDelayMs), init?.signal ?? undefined);
        continue;
      }

      return response;
    } catch (error) {
      // Don't retry abort errors
      if (error instanceof DOMException && error.name === "AbortError") {
        throw error;
      }

      if (attempt < maxRetries) {
        console.warn(
          `Attempt ${attempt + 1}/${maxRetries + 1} failed with error for ${url}:`,
          error,
        );
        await sleep(getDelay(attempt, baseDelayMs), init?.signal ?? undefined);
        continue;
      }

      throw error;
    }
  }
}
