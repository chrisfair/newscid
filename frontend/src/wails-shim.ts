// frontend/src/wails-shim.ts
// Transparent RPC & Event bridge to Go backend for Android APK WebView and Web/PWA mode

declare global {
  interface Window {
    go?: any
    runtime?: any
  }
}

export function initWailsShim() {
  if (typeof window === 'undefined') return

  // If Wails desktop desktop binding is not injected, initialize HTTP RPC proxy
  if (!window.go) {
    let reqId = 1
    const appProxy = new Proxy({}, {
      get(_target, propKey: string) {
        return async (...args: any[]) => {
          const currentId = reqId++
          const res = await fetch('/api/rpc', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              id: currentId,
              method: propKey,
              params: args,
            }),
          })

          if (!res.ok) {
            throw new Error(`RPC HTTP error ${res.status}: ${res.statusText}`)
          }

          const data = await res.json()
          if (data.error) {
            throw new Error(data.error)
          }
          return data.result
        }
      },
    })

    window.go = {
      main: {
        App: appProxy,
      },
    }
  }

  // If Wails desktop runtime is not injected, initialize web runtime
  if (!window.runtime) {
    const eventListeners: Record<string, ((...args: any[]) => void)[]> = {}

    if (typeof EventSource !== 'undefined') {
      try {
        const es = new EventSource('/api/events')
        es.addEventListener('wails', (e: MessageEvent) => {
          try {
            const payload = JSON.parse(e.data)
            const eventName = payload.name
            const data = payload.data || []
            if (eventName && eventListeners[eventName]) {
              for (const cb of eventListeners[eventName]) {
                cb(...data)
              }
            }
          } catch (err) {
            console.error('Failed to parse SSE event:', err)
          }
        })
      } catch (err) {
        console.warn('SSE event stream not available:', err)
      }
    }

    window.runtime = {
      EventsOn(eventName: string, callback: (...args: any[]) => void) {
        if (!eventListeners[eventName]) {
          eventListeners[eventName] = []
        }
        eventListeners[eventName].push(callback)
        return () => {
          if (eventListeners[eventName]) {
            eventListeners[eventName] = eventListeners[eventName].filter(cb => cb !== callback)
          }
        }
      },
      EventsOff(eventName: string, ..._additional: string[]) {
        delete eventListeners[eventName]
      },
      EventsOnce(eventName: string, callback: (...args: any[]) => void) {
        const onceWrapper = (...args: any[]) => {
          this.EventsOff(eventName)
          callback(...args)
        }
        this.EventsOn(eventName, onceWrapper)
      },
      EventsEmit(eventName: string, ...data: any[]) {
        fetch('/api/rpc', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ method: 'emitEvent', params: [eventName, ...data] }),
        }).catch(() => {})
      },
      WindowSetTitle(title: string) {
        document.title = title
      },
      LogInfo(msg: string) {
        console.log('[Go Backend]', msg)
      },
      LogError(msg: string) {
        console.error('[Go Backend]', msg)
      },
    }
  }
}
