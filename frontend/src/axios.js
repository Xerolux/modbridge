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

// A single transient 401 (e.g. during a parallel request burst) used to kick the
// user out repeatedly and reset auth state multiple times. This gate coalesces
// concurrent 401s into one redirect + one reset for ~1s.
let isHandlingUnauthorized = false;

axios.interceptors.response.use(
    response => response,
    async error => {
        const originalRequest = error.config;

        if (!originalRequest || originalRequest.skipAuth) {
            return Promise.reject(error);
        }

        // Handle authentication errors globally — but dedup the redirect so a
        // burst of parallel 401s only triggers one reset/redirect.
        if (error.response && error.response.status === 401) {
            if (!isHandlingUnauthorized) {
                isHandlingUnauthorized = true;
                emitUnauthorized();
                if (window.location.hash !== '#/login') {
                    window.location.hash = '#/login';
                }
                setTimeout(() => { isHandlingUnauthorized = false; }, 1000);
            }
            return Promise.reject(error);
        }

        // 403 — surface a toast; do NOT redirect (the user may still be allowed
        // elsewhere). Layout.vue listens and shows a PrimeVue toast.
        if (error.response && error.response.status === 403) {
            window.dispatchEvent(new CustomEvent('app:forbidden'));
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
