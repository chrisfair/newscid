import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import HelpModal from './HelpModal.vue'
import * as runtime from '../../wailsjs/runtime/runtime'

vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}))

describe('HelpModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders topics list and initial Getting Started section', () => {
    const wrapper = mount(HelpModal)

    expect(wrapper.text()).toContain('NewScid User Guide & Online Help')
    expect(wrapper.text()).toContain('Getting Started')
    expect(wrapper.text()).toContain('Databases & PGN')
    expect(wrapper.text()).toContain('Player Personas & Aliases')
    expect(wrapper.text()).toContain('Online Account Sync')
    expect(wrapper.text()).toContain('AI Player Simulation Bot')
    expect(wrapper.text()).toContain('Welcome to NewScid')
  })

  it('filters topics by search query', async () => {
    const wrapper = mount(HelpModal)

    const searchInput = wrapper.find('input[placeholder="Search help topics..."]')
    await searchInput.setValue('Persona')

    expect(wrapper.text()).toContain('Player Personas & Aliases')
    expect(wrapper.text()).not.toContain('Scorebook & Notation Styles')
  })

  it('switches active topic when clicking on sidebar topic button', async () => {
    const wrapper = mount(HelpModal)

    const topicButtons = wrapper.findAll('.w-72 button')
    const personaBtn = topicButtons.find(b => b.text().includes('Player Personas'))
    expect(personaBtn).toBeDefined()
    await personaBtn!.trigger('click')

    expect(wrapper.text()).toContain('Multi-Alias Consolidation')
    expect(wrapper.text()).toContain('Dedicated Persona Databases (.si4)')
  })

  it('invokes BrowserOpenURL when clicking Web Documentation', async () => {
    const wrapper = mount(HelpModal)

    const webBtn = wrapper.findAll('button').find(b => b.text().includes('Web Documentation'))
    expect(webBtn).toBeDefined()
    await webBtn!.trigger('click')

    expect(runtime.BrowserOpenURL).toHaveBeenCalledWith(
      expect.stringContaining('github.com/christopherpatrickfair/newscid')
    )
  })

  it('emits close when X button is clicked', async () => {
    const wrapper = mount(HelpModal)

    const closeBtn = wrapper.findAll('button').find(b => b.find('svg.lucide-x').exists() || b.html().includes('lucide-x'))
    expect(closeBtn).toBeDefined()
    await closeBtn!.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
