import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ECOBrowserModal from './ECOBrowserModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetECOSections: vi.fn().mockResolvedValue([
    { letter: 'A', title: 'Flank Openings', description: 'English, Reti', count: 120 },
    { letter: 'B', title: 'Semi-Open Games', description: 'Sicilian, Caro-Kann', count: 350 },
    { letter: 'C', title: 'Open Games & French', description: 'Ruy Lopez, Italian, French', count: 400 },
    { letter: 'D', title: 'Closed Games', description: 'Queen\'s Gambit', count: 280 },
    { letter: 'E', title: 'Indian Defenses', description: 'King\'s Indian', count: 210 },
  ]),
  SearchECO: vi.fn().mockImplementation((section: string, query: string) => {
    const list = [
      { code: 'B20', name: 'Sicilian Defense', moves: '1. e4 c5' },
      { code: 'B90', name: 'Sicilian, Najdorf', moves: '1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6' },
      { code: 'C60', name: 'Ruy Lopez', moves: '1. e4 e5 2. Nf3 Nc6 3. Bb5' },
    ]
    return Promise.resolve(list.filter(e => {
      if (section && section !== 'ALL' && !e.code.startsWith(section)) return false
      if (query && !e.name.toLowerCase().includes(query.toLowerCase()) && !e.code.toLowerCase().includes(query.toLowerCase())) return false
      return true
    }))
  }),
}))

describe('ECOBrowserModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal header, section tabs, and initial entries list', async () => {
    const wrapper = mount(ECOBrowserModal)
    await flushPromises()

    expect(wrapper.text()).toContain('ECO Opening Classification Browser')
    expect(wrapper.text()).toContain('All Openings')
    expect(wrapper.text()).toContain('Sicilian Defense')
    expect(wrapper.text()).toContain('Ruy Lopez')
  })

  it('filters entries when section tab is clicked', async () => {
    const wrapper = mount(ECOBrowserModal)
    await flushPromises()

    const sectionButtons = wrapper.findAll('.p-3 button')
    const bButton = sectionButtons.find(b => b.text().includes('B'))
    expect(bButton).toBeDefined()
    await bButton!.trigger('click')
    await flushPromises()

    expect(AppApi.SearchECO).toHaveBeenCalledWith('B', '', 150)
  })

  it('filters entries by search query input', async () => {
    const wrapper = mount(ECOBrowserModal)
    await flushPromises()

    const searchInput = wrapper.find('input[placeholder="Search ECO code, name, or moves..."]')
    await searchInput.setValue('Najdorf')
    await flushPromises()

    expect(AppApi.SearchECO).toHaveBeenCalledWith('ALL', 'Najdorf', 150)
  })

  it('emits playOpening and close when Play Line on Board is clicked', async () => {
    const wrapper = mount(ECOBrowserModal)
    await flushPromises()

    const playBtn = wrapper.findAll('button').find(b => b.text().includes('Play Line on Board'))
    expect(playBtn).toBeDefined()
    await playBtn!.trigger('click')

    expect(wrapper.emitted('playOpening')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits searchEcoInDb and close when Search Database is clicked', async () => {
    const wrapper = mount(ECOBrowserModal)
    await flushPromises()

    const searchDbBtn = wrapper.findAll('button').find(b => b.text().includes('Search Database'))
    expect(searchDbBtn).toBeDefined()
    await searchDbBtn!.trigger('click')

    expect(wrapper.emitted('searchEcoInDb')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits close when X button is clicked', async () => {
    const wrapper = mount(ECOBrowserModal)
    await flushPromises()

    const closeBtn = wrapper.findAll('button').find(b => b.find('svg.lucide-x').exists() || b.html().includes('lucide-x'))
    expect(closeBtn).toBeDefined()
    await closeBtn!.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
