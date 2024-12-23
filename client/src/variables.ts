
export const MEDIA_SERVER = import.meta.env.VITE_MEDIA_SERVER
export const MEDIA_SERVER_WS = import.meta.env.VITE_MEDIA_SERVER_WS

export const MEDIA_SERVER_STUN = import.meta.env.VITE_MEDIA_SERVER_STUN

console.log("[ENV MEDIA_SERVER]", MEDIA_SERVER)
console.log("[MEDIA_SERVER_WS]", MEDIA_SERVER_WS)
console.log("[MEDIA_SERVER_STUN]", MEDIA_SERVER_STUN)


export const IDENTITY_SERVER = `${MEDIA_SERVER}/identity`
console.log("[IDENTITY_SERVER]", IDENTITY_SERVER)

export const IDENTITY_SERVER_VERIFY_TOKEN = `${IDENTITY_SERVER}/token-verify`
export const IDENTITY_SERVER_SIGNIN = `${IDENTITY_SERVER}/sign-in`
