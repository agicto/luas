/**
 * Object utility functions
 * Common operations for object manipulation
 */

/**
 * Deep clones an object
 * Uses the native structuredClone where available, with a basic fallback
 * @param obj - Object to clone
 */
export function deepClone<T>(obj: T): T {
  if (typeof structuredClone === 'function') {
    return structuredClone(obj);
  }

  // Basic fallback for environments without structuredClone
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  
  if (Array.isArray(obj)) {
    return obj.map(item => deepClone(item)) as unknown as T;
  }

  const clonedObj: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    clonedObj[key] = deepClone(value);
  }

  return clonedObj as T;
}

/**
 * Safely gets a nested property from an object using a path string
 * @param obj - The object to get the value from
 * @param path - Path to the property (e.g., 'user.address.city')
 * @param defaultValue - Default value if path doesn't exist
 */
export function getNestedValue<T, D = undefined>(
  obj: Record<string, unknown> | null | undefined,
  path: string,
  defaultValue?: D
): T | D {
  if (!obj) return defaultValue as D;
  
  const keys = path.split('.');
  let result: unknown = obj;
  
  for (const key of keys) {
    if (result === undefined || result === null) {
      return defaultValue as D;
    }

    if (typeof result !== 'object') {
      return defaultValue as D;
    }

    result = (result as Record<string, unknown>)[key];
  }
  
  return (result === undefined) ? (defaultValue as D) : (result as T);
}

/**
 * Creates a new object with only the specified keys
 * @param obj - Original object
 * @param keys - Keys to include in the new object
 */
export function pick<T extends object, K extends keyof T>(
  obj: T, 
  keys: K[]
): Pick<T, K> {
  const result = {} as Pick<T, K>;
  keys.forEach(key => {
    if (obj && Object.prototype.hasOwnProperty.call(obj, key)) {
      result[key] = obj[key];
    }
  });
  return result;
}

/**
 * Creates a new object excluding the specified keys
 * @param obj - Original object
 * @param keys - Keys to exclude from the new object
 */
export function omit<T extends object, K extends keyof T>(
  obj: T, 
  keys: K[]
): Omit<T, K> {
  const result = { ...obj } as Omit<T, K> & Partial<T>;
  keys.forEach(key => {
    Reflect.deleteProperty(result, key);
  });
  return result;
}

/**
 * Checks if an object is empty
 * @param obj - Object to check
 */
export function isEmptyObject(obj: object | null | undefined): boolean {
  if (!obj) return true;
  return Object.keys(obj).length === 0;
}

/**
 * Merges objects deeply
 * @param target - Target object
 * @param sources - Source objects
 */
export function deepMerge<T extends object>(target: T, ...sources: object[]): T {
  if (!sources.length) return target;
  const source = sources.shift();

  if (isObject(target) && isObject(source)) {
    const mutableTarget = target as Record<string, unknown>;
    const sourceRecord = source as Record<string, unknown>;

    Object.keys(sourceRecord).forEach(key => {
      const sourceValue = sourceRecord[key];
      const targetValue = mutableTarget[key];

      if (isObject(sourceValue)) {
        if (!targetValue) {
          mutableTarget[key] = deepClone(sourceValue);
        } else {
          mutableTarget[key] = deepMerge(targetValue as Record<string, unknown>, sourceValue);
        }
      } else {
        mutableTarget[key] = sourceValue;
      }
    });
  }

  return deepMerge(target, ...sources);
}

/**
 * Deep freezes an object
 * @param obj - Object to freeze
 */
export function deepFreeze<T extends object>(obj: T): T {
  Object.freeze(obj);
  Object.getOwnPropertyNames(obj).forEach(prop => {
    const value = (obj as Record<string, unknown>)[prop];
    if (
      value !== null &&
      (typeof value === 'object' || typeof value === 'function') &&
      !Object.isFrozen(value)
    ) {
      deepFreeze(value);
    }
  });
  return obj;
}

/**
 * Helper: Checks if value is a plain object
 */
function isObject(item: unknown): item is Record<string, unknown> {
  return item !== null && typeof item === 'object' && !Array.isArray(item);
}
