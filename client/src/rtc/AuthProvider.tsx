import { PropsWithChildren, createContext, useState, useContext, useCallback, useEffect, useMemo } from "react";
import { Mutex } from "../App";
import { IDENTITY_SERVER_SIGNIN, IDENTITY_SERVER_VERIFY_TOKEN } from "../variables";
import { useCookies } from "react-cookie";

type TokenPair = {
  accessToken: string | undefined
}

type AuthContextType = {
  tokenPair: TokenPair | undefined,
  setTokenPair: (accessToken: string | undefined) => Promise<any>
};

const AuthContext = createContext<AuthContextType>(undefined as never)

export function useAuth() {
  const { tokenPair, setTokenPair } = useContext(AuthContext)

  const signIn = async (params: { username: string, password: string }) => {
    const { username, password } = params

    const resp = await fetch(IDENTITY_SERVER_SIGNIN, {
      method: "POST",
      body: JSON.stringify({
        username,
        password,
      }),
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      }
    })

    if (resp.status !== StatusCode.OK) {
      throw resp
    }

    const { access_token = undefined } = await resp.json()

    if (!access_token) {
      throw new Response(JSON.stringify({
        message: "Something goes wrong"
      }))
    }

    await setTokenPair(access_token)

    return resp
  }

  const signOut = async () => {
    // TODO: Add server request

    setTokenPair(undefined)
  }

  const authenticated = useMemo<boolean>(() => {
    if (!tokenPair) return false

    const { accessToken = undefined } = tokenPair

    if (!accessToken) return false

    return true
  }, [tokenPair])

  return { authenticated, signIn, signOut }
}


const onRequestMutex = new Mutex()
const onUnauthorizedMutex = new Mutex()

type FetchRequestInit = RequestInit & {
}

enum StatusCode {
  OK = 200,
  Created = 201,
  Unauthorized = 401,
}

export function useAuthorizedFetch() {
  const { tokenPair, setTokenPair } = useContext(AuthContext)

  const __fetch = useCallback(async (url: string, init?: FetchRequestInit) => {
    if (!tokenPair || !setTokenPair) {
      throw Error("Empty identity")
    }
    if (!tokenPair?.accessToken) {
      throw Error(`Trying access authorized fetch by unauthorized identity request url: ${url}, init: ${init}`)
    }

    if (!init) {
      init = { headers: {} }
    } else {
      if (!init.headers) {
        init.headers = {}
        init.credentials = 'include'
      }
    }

    // @ts-ignore
    init.headers['Authorization'] = `Bearer ${tokenPair?.accessToken}`

    const unlock = await onRequestMutex.lock()

    // NOTE: When identity is unauthorized skip request.  
    // In feature when identity will be valid retry it by useCallback reactivity
    if (onUnauthorizedMutex.isLocked()) {
      console.warn(`Trying send authorized by unauthorized identity request url: ${url} , init: ${init}`)
      unlock()
      return
    }

    try {
      const resp = await fetch(url, init).catch()

      switch (resp.status) {
        case StatusCode.Unauthorized:
          fetch(`${IDENTITY_SERVER_VERIFY_TOKEN}`, {
            method: 'POST',
            credentials: 'include',
          }).then(r => r.json())
            .then(r => {
              const { access_token = undefined } = r
              setTokenPair(access_token)
            })
            .catch(() => setTokenPair(undefined,))
      }

      return resp
    } finally {
      unlock()
    }
  }, [tokenPair, setTokenPair])

  return {
    fetch: __fetch,
  }
}

const __COOKIE_ACCESS_TOKEN = "__access_token"

export function AuthContextProvider({ children }: PropsWithChildren<{}>) {
  const [tokenPair, __setTokenPair] = useState<TokenPair>()
  const [cookies, setCookie, removeCookie] = useCookies([__COOKIE_ACCESS_TOKEN,])

  useEffect(() => {
    console.log("tokenPair", tokenPair)
  }, [tokenPair])

  useEffect(() => {
    __setTokenPair({ accessToken: cookies[__COOKIE_ACCESS_TOKEN] || undefined, })
  }, [])

  const setTokenPair = async (accessToken: string | undefined) => {
    if (!accessToken) {
      console.warn(`Reset token pair identity accessToken: ${accessToken}`)
      removeCookie(__COOKIE_ACCESS_TOKEN)
      document.cookie = ""
      __setTokenPair(undefined)
      return
    }

    const date = new Date()
    date.setFullYear(date.getFullYear() + 100)

    setCookie(__COOKIE_ACCESS_TOKEN, accessToken, {
      expires: date,
    })

    __setTokenPair({ accessToken, })

    console.log("Refresh token pair", accessToken)
  }

  return (
    <AuthContext.Provider value={{ tokenPair, setTokenPair, }}>
      {children}
    </AuthContext.Provider>
  )
}

