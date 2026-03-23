/// <reference types="vite/client" />

interface ImportMetaEnv { readonly VITE_BK_AUTH_URL: string }

interface ImportMeta { readonly env: ImportMetaEnv }

declare module '*.css' {
  const css: string;
  export default css;
}

declare module '*.png' {
  const css: string;
  export default png;
}

declare module '*.js' {
  const css: string;
  export default js;
}

declare module '@blueking/login-modal' {
  export function showLoginModal(params: { loginUrl: string }): void;
}

declare interface Window {
  BKANALYSIS?: { init: (params: { siteName: string }) => void }
  BK_AUTH_URL: string
  BK_SITE_PATH: string
  BK_STATIC_URL: string
}

declare global {
  var BK_AUTH_URL: string;
  var BK_SITE_PATH: string;
  var BK_STATIC_URL: string;
}
