import axios from 'axios';
import { emitUnauthorized } from './utils/authEvents.js';

axios.defaults.withCredentials = true;

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

axios.interceptors.request.use(config => {
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
    error => {
        if (error.response && error.response.status === 401) {
            emitUnauthorized();
            if (window.location.hash !== '#/login') {
                window.location.replace('/#/login');
            }
        }
        return Promise.reject(error);
    }
);

export default axios;
