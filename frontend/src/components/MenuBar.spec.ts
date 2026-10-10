import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MenuBar from './MenuBar.vue'

describe('MenuBar.vue', () => {
  const defaultProps = {
    dbName: 'TestDB',
    pieceSets: [{ id: 'cburnett', name: 'CBurnett', isScidNative: false, description: 'Standard' }],
    boardThemes: [{ id: 'classic-wood', name: 'Classic Wood', light: '#f0d9b5', dark: '#b58863', highlight: '#cdd26a' }],
    currentPieceSet: 'cburnett',
    currentThemeId: 'classic-wood',
    trialMode: false,
    isAnalyzing: false,
    soundEnabled: true,
    blindfold: false,
    threatMode: false,
    showEvalBar: true,
  }

  it('renders menu bar with top-level menus', () => {
    const wrapper = mount(MenuBar, { props: defaultProps })
    const text = wrapper.text()
    expect(text).toContain('File')
    expect(text).toContain('Edit')
    expect(text).toContain('Game')
    expect(text).toContain('Search')
    expect(text).toContain('Tools')
    expect(text).toContain('Play')
    expect(text).toContain('Options')
    expect(text).toContain('Help')
  })

  it('opens Play menu and emits chess960 when Fischer Random is clicked', async () => {
    const wrapper = mount(MenuBar, { props: defaultProps })
    const playBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'Play')
    expect(playBtn).toBeDefined()
    await playBtn!.trigger('click')

    const chess960Btn = wrapper.findAll('button').find((b) => b.text().includes('Chess960'))
    expect(chess960Btn).toBeDefined()
    await chess960Btn!.trigger('click')

    expect(wrapper.emitted('chess960')).toBeTruthy()
  })

  it('opens Play menu and emits coordinateTrainer when clicked', async () => {
    const wrapper = mount(MenuBar, { props: defaultProps })
    const playBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'Play')
    await playBtn!.trigger('click')

    const trainerBtn = wrapper.findAll('button').find((b) => b.text().includes('Coordinate & Vision Trainer'))
    expect(trainerBtn).toBeDefined()
    await trainerBtn!.trigger('click')

    expect(wrapper.emitted('coordinateTrainer')).toBeTruthy()
  })

  it('opens Options menu and toggles sound, eval bar, threat mode, and blindfold', async () => {
    const wrapper = mount(MenuBar, { props: defaultProps })
    const optionsBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'Options')
    await optionsBtn!.trigger('click')

    // Sound toggle
    const soundBtn = wrapper.findAll('button').find((b) => b.text().includes('Sound Effects'))
    expect(soundBtn).toBeDefined()
    await soundBtn!.trigger('click')
    expect(wrapper.emitted('toggleSound')).toBeTruthy()

    // Reopen options
    await optionsBtn!.trigger('click')
    const evalBtn = wrapper.findAll('button').find((b) => b.text().includes('Evaluation Gauge'))
    expect(evalBtn).toBeDefined()
    await evalBtn!.trigger('click')
    expect(wrapper.emitted('toggleEvalBar')).toBeTruthy()

    // Threat mode
    await optionsBtn!.trigger('click')
    const threatBtn = wrapper.findAll('button').find((b) => b.text().includes('Threat / Spy Highlights'))
    expect(threatBtn).toBeDefined()
    await threatBtn!.trigger('click')
    expect(wrapper.emitted('toggleThreatMode')).toBeTruthy()

    // Blindfold mode
    await optionsBtn!.trigger('click')
    const blindfoldBtn = wrapper.findAll('button').find((b) => b.text().includes('Blindfold Mode'))
    expect(blindfoldBtn).toBeDefined()
    await blindfoldBtn!.trigger('click')
    expect(wrapper.emitted('toggleBlindfold')).toBeTruthy()
  })
})
