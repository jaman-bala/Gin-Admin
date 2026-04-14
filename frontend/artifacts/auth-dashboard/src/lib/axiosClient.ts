import axios from "axios";
import { useAuthStore } from "../store/authStore";

// Константы
const REFRESH_URL = '/api/v1/auth/refresh';
const LOGIN_PATH = '/login';

// Инстанс для auth-запросов (чтобы избежать цикла в interceptor)
const authAxios = axios.create({ baseURL: "/" });

export const axiosClient = axios.create({
  baseURL: "/",
});

axiosClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

let isRefreshing = false;
let failedQueue: Array<{ resolve: (value?: unknown) => void; reject: (reason?: any) => void }> = [];

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

const redirectToLogin = () => {
  useAuthStore.getState().clearAuth();
  window.location.href = LOGIN_PATH;
};

axiosClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (originalRequest.url === REFRESH_URL) {
        redirectToLogin();
        return Promise.reject(error);
      }

      if (isRefreshing) {
        return new Promise(function (resolve, reject) {
          failedQueue.push({ resolve, reject });
        })
          .then((token) => {
            originalRequest.headers.Authorization = `Bearer ${token}`;
            return axiosClient(originalRequest);
          })
          .catch((err) => {
            return Promise.reject(err);
          });
      }

      originalRequest._retry = true;
      isRefreshing = true;
      const refreshToken = useAuthStore.getState().refreshToken;

      if (!refreshToken) {
        redirectToLogin();
        return Promise.reject(error);
      }

      try {
        const { data } = await authAxios.post(REFRESH_URL, { refresh_token: refreshToken });
        const newAccessToken = data.access_token;
        const newRefreshToken = data.refresh_token ?? refreshToken; // Сохраняем новый refresh токен
        
        // Optionally update user if returned
        if (data.user) {
          useAuthStore.getState().setAuth(data.user, newAccessToken, newRefreshToken);
        } else {
          useAuthStore.getState().setTokens(newAccessToken, newRefreshToken);
        }

        processQueue(null, newAccessToken);
        isRefreshing = false; // <-- Правильное место: после processQueue
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        return axiosClient(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError as Error, null);
        isRefreshing = false;
        redirectToLogin();
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);
