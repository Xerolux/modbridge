import axios from 'axios';
import { emitUnauthorized } from './utils/authEvents.js';

axios.defaults.withCredentials = true;
axios.defaults.timeout = 30000;
axios.defaults.headers.common['X-Requested-With'] = 'XMLHttpRequest';

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

axios.interceptors.request.use(config => {
    if (config.skipAuth) {
        return config;
    }
    const csrfToken = getCookie('csrf_token');
    if (csrfToken) {
        config.headers['X-CSRF-Token'] = csrfToken;
    }
    return config;
}, error => {
    return Promise.reject(error);
});

axios.interceptors.response.use(
    response => response,
    async error => {
        const originalRequest = error.config;

        if (!originalRequest || originalRequest.skipAuth) {
            return Promise.reject(error);
        }

        // Handle authentication errors globally.
        if (error.response && error.response.status === 401) {
            emitUnauthorized();
            if (window.location.hash !== '#/login') {
                window.location.hash = '#/login';
            }
            return Promise.reject(error);
        }

        // Retry idempotent requests on transient failures.
        const isRetryable = !originalRequest._retryCount &&
            (!error.response || error.response.status >= 503 || error.response.status === 429);
        if (isRetryable && ['get', 'head', 'options', 'put', 'delete'].includes(originalRequest.method?.toLowerCase())) {
            originalRequest._retryCount = (originalRequest._retryCount || 0) + 1;
            await new Promise(resolve => setTimeout(resolve, 500));
            return axios(originalRequest);
        }

        return Promise.reject(error);
    }
);

export default axios;
