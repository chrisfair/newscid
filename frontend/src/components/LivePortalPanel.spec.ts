import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LivePortalPanel from './LivePortalPanel.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
}))

vi.mock('../../wailsjs/go/main/App', () => ({
  LaunchPortalWindow: vi.fn().mockResolvedValue(undefined),
  GetLivePortalCaptureStatus: vi.fn().mockResolvedValue({
    enabled: true,
    autoLoadToBoard: true,
    watcherActive: true,
    intervalSeconds: 25,
    lichessUsername: 'PortalHero',
    chessComUsername: 'PortalHeroCC',
    targetDatabasePath: '/tmp/My_Games',
    targetDatabaseName: 'My_Games',
    sessionCapturedCount: 2,
    lastCapturedGameId: 14,
    lastCapturedGameSummary: 'PortalHero vs Challenger (1-0) • Rated Blitz',
  }),
  ConfigureLivePortalAutoCapture: vi.fn().mockResolvedValue({
    enabled: true,
    autoLoadToBoard: true,
    watcherActive: true,
    lichessUsername: 'PortalHero',
    chessComUsername: 'PortalHeroCC',
    targetDatabaseName: 'My_Games',
    sessionCapturedCount: 2,
  }),
  CheckAndCaptureLivePortalGames: vi.fn().mockResolvedValue({
    newGamesImported: 1,
    lichessImported: 1,
    chessComImported: 0,
    duplicatesSkipped: 3,
    latestGameId: 15,
    latestGameSummary: 'PortalHero vs GM_Test (1-0) • Rated Blitz',
    targetDatabaseName: 'My_Games',
    autoLoadToBoard: true,
  }),
  CaptureOnlineGameFromURLOrPGN: vi.fn().mockResolvedValue({
    newGamesImported: 1,
    latestGameId: 16,
    latestGameSummary: 'PortalHero vs Opponent (0-1) • Live Chess',
    targetDatabaseName: 'My_Games',
    autoLoadToBoard: true,
  }),
}))

describe('LivePortalPanel.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders Live Auto-Capture status, configured usernames, and target database', async () => {
    const wrapper = mount(LivePortalPanel, {
      props: {
        currentFen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
      }
    })
    await flushPromises()

    expect(AppApi.GetLivePortalCaptureStatus).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Auto-Capture to DB: ON')
    expect(wrapper.text()).toContain('My_Games')
    expect(wrapper.text()).toContain('@PortalHero')
    expect(wrapper.text()).toContain('@PortalHeroCC')
    expect(wrapper.text()).toContain('+2 auto-saved')
  })

  it('triggers CheckAndCaptureLivePortalGames when clicking Capture Latest Now', async () => {
    const wrapper = mount(LivePortalPanel, {
      props: {
        currentFen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
      }
    })
    await flushPromises()

    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('Capture Latest Now'))
    expect(checkBtn).toBeDefined()
    await checkBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.CheckAndCaptureLivePortalGames).toHaveBeenCalled()
    expect(wrapper.emitted('gameCaptured')).toBeTruthy()
  })

  it('captures a pasted Lichess or Chess.com game URL directly into the database', async () => {
    const wrapper = mount(LivePortalPanel, {
      props: {
        currentFen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
      }
    })
    await flushPromises()

    const urlInput = wrapper.findAll('input').find(i => i.attributes('placeholder')?.includes('Paste any Lichess/Chess.com game URL'))
    expect(urlInput).toBeDefined()
    await urlInput!.setValue('https://lichess.org/AbCdEfGh/black')

    const captureBtn = wrapper.findAll('button').find(b => b.text().includes('Capture Game to DB'))
    expect(captureBtn).toBeDefined()
    await captureBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.CaptureOnlineGameFromURLOrPGN).toHaveBeenCalledWith('https://lichess.org/AbCdEfGh/black')
    expect(wrapper.emitted('gameCaptured')).toBeTruthy()
  })
})
