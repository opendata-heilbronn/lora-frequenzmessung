import {reactive, readonly} from "vue";
import api from "../api.ts";

type UserInfo = {
    id?: string;
    username?: string;
    token?: string;
}

type Claims = {
    sub: string;
    username: string;
    exp: number;
}

const userStore: UserInfo = reactive({});

export const user = readonly(userStore);

export const setToken = (token: string): void => {
    userStore.token = token;

    const claims = parseTokenClaims(token);
    if (!claims) {
        return;
    }

    userStore.id = claims.sub;
    userStore.username = claims.username;
};

function parseTokenClaims(token: string): Claims | undefined {
    const parts = token.split('.');
    if (parts.length != 3) {
        return undefined;
    }

    return JSON.parse(base64Decode(parts[1]));
}

function base64Decode(str: string): string {
    return decodeURIComponent(atob(str).split('').map(function (c) {
        return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
    }).join(''));
}

export function loadAndRefreshToken() {
    const token = localStorage.getItem('token');
    if (!token) {
        return;
    }

    setToken(token);

    const claims = parseTokenClaims(token);
    if (!claims) {
        refresh();
        return;
    }
    const refreshIn = (claims.exp * 1000 - Date.now()) / 2;
    setTimeout(refresh, refreshIn);
}

async function refresh() {
    try {
        const response = await api.post<{ token: string }>('/auth/refresh');
        if (response.status !== 200) {
            return;
        }

        localStorage.setItem('token', response.data.token)
        setToken(response.data.token);

        const claims = parseTokenClaims(response.data.token);
        if (!claims) {
            return;
        }

        const refreshIn = (claims.exp * 1000 - Date.now()) / 2;
        setTimeout(refresh, refreshIn);
    } catch (e) {
        return;
    }
}