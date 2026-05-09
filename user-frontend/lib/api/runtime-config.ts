/**
 * 运行时配置模块
 * 
 * 提供运行时获取 API 配置的能力，替代构建时注入的 NEXT_PUBLIC_API_URL。
 * 这使得前端可以在不同环境中使用不同的 API 地址，而无需重新构建。
 */

export interface RuntimeConfig {
  /** API 基础 URL */
  apiUrl: string;
  /** 后端版本号 */
  version: string;
}

// 缓存运行时配置，避免重复请求
let runtimeConfigCache: RuntimeConfig | null = null;

// 缓存 Promise，防止并发请求
let configPromise: Promise<RuntimeConfig> | null = null;

/**
 * 获取运行时配置
 * 
 * 首次调用会从后端 /api/v1/public/config 获取配置，
 * 后续调用返回缓存的配置。
 * 
 * @returns 运行时配置对象
 */
export async function getRuntimeConfig(): Promise<RuntimeConfig> {
  // 如果已有缓存，直接返回
  if (runtimeConfigCache) {
    return runtimeConfigCache;
  }

  // 如果正在请求中，返回现有的 Promise
  if (configPromise) {
    return configPromise;
  }

  // 创建新的请求 Promise
  configPromise = fetchRuntimeConfig();

  try {
    runtimeConfigCache = await configPromise;
    return runtimeConfigCache;
  } catch (error) {
    // 请求失败时清除 Promise，允许重试
    configPromise = null;
    throw error;
  }
}

/**
 * 从后端获取运行时配置
 */
async function fetchRuntimeConfig(): Promise<RuntimeConfig> {
  try {
    // 在服务端渲染时，使用环境变量或默认值
    if (typeof window === 'undefined') {
      return getDefaultConfig();
    }

    const response = await fetch('/api/v1/public/config', {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
      // 设置较短的超时时间，避免阻塞页面加载
      signal: AbortSignal.timeout(5000),
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch runtime config: ${response.status}`);
    }

    const data = await response.json();
    
    return {
      apiUrl: data.apiUrl || '',
      version: data.version || 'unknown',
    };
  } catch (error) {
    console.warn('[RuntimeConfig] Failed to load runtime config:', error);
    // 返回默认配置作为回退
    return getDefaultConfig();
  }
}

/**
 * 获取默认配置
 * 
 * 当运行时配置获取失败时使用此配置。
 * 优先使用构建时环境变量，最后使用硬编码默认值。
 */
function getDefaultConfig(): RuntimeConfig {
  return {
    apiUrl: process.env.NEXT_PUBLIC_API_URL || '',
    version: 'unknown',
  };
}

/**
 * 获取 API 基础 URL
 * 
 * 如果运行时配置已加载，使用运行时配置的值；
 * 否则使用构建时环境变量或默认值。
 * 
 * @returns API 基础 URL
 */
export function getApiBaseUrl(): string {
  // 如果缓存存在，使用缓存值
  if (runtimeConfigCache?.apiUrl) {
    return runtimeConfigCache.apiUrl;
  }

  // 否则使用环境变量或默认值
  return process.env.NEXT_PUBLIC_API_URL || '';
}

/**
 * 清除运行时配置缓存
 * 
 * 用于测试或需要强制刷新配置的场景。
 */
export function clearRuntimeConfigCache(): void {
  runtimeConfigCache = null;
  configPromise = null;
}

/**
 * 初始化运行时配置
 * 
 * 应在应用启动时调用，确保配置已加载。
 * 可以在 layout 或 _app.tsx 中使用。
 * 
 * @example
 * ```tsx
 * // app/layout.tsx 或 pages/_app.tsx
 * useEffect(() => {
 *   initRuntimeConfig();
 * }, []);
 * ```
 */
export function initRuntimeConfig(): void {
  // 只在客户端执行
  if (typeof window === 'undefined') {
    return;
  }

  // 异步加载配置，不阻塞页面渲染
  getRuntimeConfig().catch((error) => {
    console.error('[RuntimeConfig] Failed to initialize:', error);
  });
}
