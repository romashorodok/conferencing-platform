import { PropsWithChildren, createContext, useState, useContext, useCallback, useEffect, useMemo } from "react";
import { Mutex } from "../App";
import { IDENTITY_SERVER_SIGNIN, IDENTITY_SERVER_VERIFY_TOKEN } from "../variables";
import { useCookies } from "react-cookie";
import { throws } from "assert";

type TokenPair = {
  accessToken: string | undefined
  refreshToken: string | undefined
}

type AuthContextType = {
  tokenPair: TokenPair | undefined,
  setTokenPair: (accessToken: string | undefined, refreshToken: string | undefined) => Promise<any>
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
      headers: {
        'Content-Type': 'application/json',
      }
    })

    if (resp.status !== StatusCode.OK) {
      throw resp
    }

    const { access_token = undefined, refresh_token = undefined } = await resp.json()

    if (!access_token || !refresh_token) {
      throw new Response(JSON.stringify({
        message: "Something goes wrong"
      }))
    }

    await setTokenPair(access_token, refresh_token)

    return resp
  }

  const signOut = async () => {
    // TODO: Add server request

    setTokenPair(undefined, undefined)
  }

  const authenticated = useMemo<boolean>(() => {
    if (!tokenPair) return false

    const { accessToken = undefined, refreshToken = undefined } = tokenPair

    if (!accessToken || !refreshToken) return false

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
    if (!tokenPair || !setTokenPair) return
    if (!tokenPair?.accessToken || !tokenPair.refreshToken) {
      console.warn(`Trying access authorized fetch by unauthorized identity request url: ${url}, init: ${init}`)
      return
    }

    if (!init) {
      init = { headers: {} }
    } else {
      if (!init.headers) {
        init.headers = {}
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
            headers: {
              Authorization: `Bearer ${tokenPair.refreshToken}`
            },
          }).then(r => r.json())
            .then(r => {
              const { access_token = undefined, refresh_token = undefined } = r
              setTokenPair(access_token, refresh_token)
            })
            .catch(() => setTokenPair(undefined, undefined))
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
const __COOKIE_REFRESH_TOKEN = "__refresh_token"

const AccessTokenStub = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjAzYjc2ZTM1LTE3MDktNGIxYS1hOTVjLTExOWUxOTRiZDJjNiIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsiY2FuOmpvaW4iXSwiZXhwIjoxNzE1MTA4NTE3LCJpc3MiOiIwLjAuMC4wIiwic3ViIjoibXl1c2VyMTIzNDU2MDgiLCJ0b2tlbjp1c2UiOiJhY2Nlc3NfdG9rZW4iLCJ1c2VyOmlkIjoiNDU3NDBjMDEtODA3ZC00OTRhLWI5MjgtYmNmODIyN2JmYWJmIn0.jY9RF5L7MIioqsR1AZSQv-0xKLnzrKGgMjfCx2kdxj_a99PNUZKOfR3zXLNPmomL3DM-I3EiQOdsHS-lrVPCxS0FWeIT94LB8cs8KTsWGI7t7hd-TqyyLgnEXia8ySmwZtl2umKWJVPOpcAMw4K-WPoY7sseTuZ9fHo_AhYHvVERVpEX288nGDlRCYsh45KglZqCJ25tSv38lGPvEtxGvvjnE4Wo8GITEt-oC9-JfvHSM30N9tjFBHCE9KLgGbzf3bghoj_GmOAfMJkveknITW1Nlf6nE3_Oua5xVLWoGvc0HUxSVlHgL9RH_GT7exGr2h3DxMkX7jXtJv_8ly2IMA"

const RefreshTokenStub = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVmYzQ4ODBiLTBiNzktNGZiMi05NmQ4LTIzNmViYThmZmMyYyIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsiY2FuOmpvaW4iXSwiZXhwIjoxNzQ2OTIyMTAyLCJpc3MiOiIwLjAuMC4wIiwic3ViIjoibXl1c2VyMTIzNDU2MDgiLCJ0b2tlbjp1c2UiOiJyZWZyZXNoX3Rva2VuIiwidXNlcjppZCI6IjQ1NzQwYzAxLTgwN2QtNDk0YS1iOTI4LWJjZjgyMjdiZmFiZiJ9.C6DJi5tyX5kSKAkjASq6gIqSxjIeH5skiZ2b_kJ7NX-JSiNzjUsOP1O1rOD7ti_pWOenL0Xkx36Q9_cFST4A82LhmLx27E06BumsUQLCosRbBf0oyxkphgVjemxFJOclg1-Kk9QiHvdjbA00NWGxtV4r-Ta3VP63TQPeJI5xNJp5fjEJJ8OyahTyn4iRyPP4dxne7r7EeIWc3jmYvvVJjMAkSB7YRFnosUyF3msHjq8JRa3AOBKIRz9ZU7ghvXfLOPfWZu8a1k53IIbFDiC6ercnALJ1lqmRUpWcXqWu_JlNE_mERqxJp3YXHwmbMYMWcdMCDBMFy487wQv8NJD9QA"

export function AuthContextProvider({ children }: PropsWithChildren<{}>) {
  const [tokenPair, __setTokenPair] = useState<TokenPair>()
  const [cookies, setCookie, removeCookie] = useCookies([__COOKIE_ACCESS_TOKEN, __COOKIE_REFRESH_TOKEN])

  useEffect(() => {
    console.log("tokenPair", tokenPair)
  }, [tokenPair])

  useEffect(() => {
    // const date = new Date()
    // date.setFullYear(date.getFullYear() + 100)
    // setCookie(__COOKIE_ACCESS_TOKEN, AccessTokenStub, {
    //   expires: date,
    // })
    // setCookie(__COOKIE_REFRESH_TOKEN, RefreshTokenStub, {
    //   expires: date,
    // })

    __setTokenPair({
      accessToken: cookies[__COOKIE_ACCESS_TOKEN] || undefined,
      refreshToken: cookies[__COOKIE_REFRESH_TOKEN] || undefined,
    })
    console.log(cookies)
  }, [])

  const setTokenPair = async (accessToken: string | undefined, refreshToken: string | undefined) => {
    if (!accessToken || !refreshToken) {
      console.warn(`Reset token pair identity accessToken: ${accessToken}, refreshToken: ${refreshToken}`)
      removeCookie(__COOKIE_ACCESS_TOKEN)
      removeCookie(__COOKIE_REFRESH_TOKEN)
      document.cookie = ""
      __setTokenPair(undefined)
      return
    }

    const date = new Date()
    date.setFullYear(date.getFullYear() + 100)

    setCookie(__COOKIE_ACCESS_TOKEN, accessToken, {
      expires: date,
    })
    setCookie(__COOKIE_REFRESH_TOKEN, refreshToken, {
      expires: date,
    })

    __setTokenPair({
      accessToken,
      refreshToken,
    })

    console.log("Refresh token pair", accessToken, refreshToken)
  }

  return (
    <AuthContext.Provider value={{ tokenPair, setTokenPair, }}>
      {children}
    </AuthContext.Provider>
  )
}

