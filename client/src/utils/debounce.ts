
export function debounce(callback: Function, timeoutMs: number = 1000) {
  let timeout: NodeJS.Timeout;

  return (...args: any) => {
    clearTimeout(timeout);

    timeout = setTimeout(() => callback(...args), timeoutMs);
  };
};
