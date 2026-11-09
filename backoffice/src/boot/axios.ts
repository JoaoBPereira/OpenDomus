import { defineBoot } from '#q-app';
import axios, { type AxiosInstance, type AxiosRequestHeaders } from 'axios';
import { serverConfig } from '@/composables/useServerConfig';

declare module 'vue' {
  interface ComponentCustomProperties {
    $axios: AxiosInstance;
    $api: AxiosInstance;
  }
}

// Be careful when using SSR for cross-request state pollution
// due to creating a Singleton instance here;
// If any client changes this (global) instance, it might be a
// good idea to move this instance creation inside of the
// "export default () => {}" function below (which runs individually
// for each client)
// create an instance with a sensible default; it'll be overridden by runtime config at boot
const api = axios.create({ baseURL: 'https://api.example.com' });

export default defineBoot(({ app, router }) => {
  // boot/config.ts runs first, so serverConfig.url already holds any config.json override
  axios.defaults.baseURL = serverConfig.url;
  api.defaults.baseURL = serverConfig.url;

  api.interceptors.request.use((config) => {
    try {
      const token = localStorage.getItem('jwt');
      if (token) {
        if (!config.headers) config.headers = {} as AxiosRequestHeaders;
        config.headers['Authorization'] = `Bearer ${token}`;
      }
    } catch {
      // ignore
    }
    return config;
  });

  api.interceptors.response.use(
    (r) => r,
    (error) => {
      const status = error?.response?.status;
      if (status === 401) {
        try {
          localStorage.removeItem('jwt');
          if (router && typeof router.push === 'function') {
            void router.push('/');
          }
        } catch {
          // ignore
        }
      }
      return Promise.reject(error as Error);
    }
  );
  // for use inside Vue files (Options API) through this.$axios and this.$api

  app.config.globalProperties.$axios = axios;
  // ^ ^ ^ this will allow you to use this.$axios (for Vue Options API form)
  //       so you won't necessarily have to import axios in each vue file

  app.config.globalProperties.$api = api;
  // ^ ^ ^ this will allow you to use this.$api (for Vue Options API form)
  //       so you can easily perform requests against your app's API
});

export { api };
