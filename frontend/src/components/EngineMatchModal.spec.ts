import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EngineMatchModal from './EngineMatchModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  IsEngineMatchRunning: vi.fn().mockResolvedValue(false),
  StopEngineMatch: vi.fn().mockResolvedValue(true),
  StartEngineMatch: vi.fn().mockImplementation((cfg: any) => {
    return Promise.resolve({
      engine1Name: cfg.engine1.name,
      engine2Name: cfg.engine2.name,
      rounds: cfg.rounds,
      engine1Score: 1.5,
      engine2Score: 0.5,
      engine1Wins: 1,
      engine2Wins: 0,
      draws: 1,
      durationMs: 800,
      games: [
        {
          round: 1,
          whiteEngine: cfg.engine1.name,
          blackEngine: cfg.engine2.name,
          result: '1-0',
          termination: 'checkmate',
          plies: 24,
          movesUci: ['e2e4', 'e7e5'],
          movesSan: ['e4', 'e5'],
          finalFen: '8/8/8/8/8/8/8/8 w - - 0 1',
          pgn: '1. e4 e5 1-0',
          durationMs: 400
        },
        {
          round: 2,
          whiteEngine: cfg.engine2.name,
          blackEngine: cfg.engine1.name,
          result: '1/2-1/2',
          termination: '3-fold repetition',
          plies: 30,
          movesUci: ['d2d4', 'd7d5'],
          movesSan: ['d4', 'd5'],
          finalFen: '8/8/8/8/8/8/8/8 w - - 0 1',
          pgn: '1. d4 d5 1/2-1/2',
          durationMs: 400
        }
      ]
    })
  })
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn()
}))

describe('EngineMatchModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal with engine inputs and default settings', async () => {
    const wrapper = mount(EngineMatchModal, {
      props: {
        discoveredEngines: [
          { name: 'Stockfish 16', cmd: '/usr/bin/stockfish', available: true, isDefault: true },
          { name: 'Crafty 25', cmd: '/usr/bin/crafty', available: true, isDefault: false }
        ],
        availableEngines: ['Stockfish 16', 'Crafty 25']
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Automated Engine vs Engine Match')
    expect(wrapper.text()).toContain('Tournament Mode')

    const inputs = wrapper.findAll('input[type="text"]')
    expect(inputs.length).toBeGreaterThanOrEqual(2)
    expect((inputs[0].element as HTMLInputElement).value).toBe('Stockfish 16')
    expect((inputs[1].element as HTMLInputElement).value).toBe('Crafty 25')
  })

  it('starts engine match, updates scores, renders completed games and emits matchFinished', async () => {
    const wrapper = mount(EngineMatchModal, {
      props: {
        discoveredEngines: [
          { name: 'Stockfish', cmd: '/usr/bin/stockfish', available: true, isDefault: true },
          { name: 'Crafty', cmd: '/usr/bin/crafty', available: true, isDefault: false }
        ],
        availableEngines: ['Stockfish', 'Crafty']
      }
    })
    await flushPromises()

    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Tournament Match'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.StartEngineMatch).toHaveBeenCalledWith(expect.objectContaining({
      rounds: 4,
      timeControlMs: 500,
      alternateColors: true,
      autoSaveToDb: true
    }))

    expect(wrapper.emitted('matchFinished')).toBeTruthy()
  })

  it('stops engine match when Stop Match button is clicked', async () => {
    vi.mocked(AppApi.IsEngineMatchRunning).mockResolvedValueOnce(true)

    const wrapper = mount(EngineMatchModal)
    await flushPromises()

    const stopBtn = wrapper.findAll('button').find(b => b.text().includes('Stop Match'))
    expect(stopBtn).toBeDefined()
    await stopBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.StopEngineMatch).toHaveBeenCalled()
  })
})
