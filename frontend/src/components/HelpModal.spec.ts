import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import HelpModal from './HelpModal.vue'

describe('HelpModal.vue', () => {
  beforeEach(() => {})

  it('renders 100% local manual topics list and initial Getting Started section with no external links', () => {
    const wrapper = mount(HelpModal)

    expect(wrapper.text()).toContain('NewScid Local Workstation Manual & Contextual Help')
    expect(wrapper.text()).toContain('100% Offline Local Docs')
    expect(wrapper.text()).toContain('Getting Started')
    expect(wrapper.text()).toContain('Workspaces & Detachable Panels')
    expect(wrapper.text()).toContain('Databases, PGN & Mega Updates')
    expect(wrapper.text()).toContain('Player Personas & Aliases')
    expect(wrapper.text()).toContain('Online Account Sync')
    expect(wrapper.text()).toContain('Fair-Play & Integrity Inspector')
    expect(wrapper.text()).toContain('Companion Workstation CLI Tools')
    expect(wrapper.text()).not.toContain('Web Documentation')
  })

  it('supports contextual help via initialTopic prop', () => {
    const wrapper = mount(HelpModal, {
      props: {
        initialTopic: 'fairplay'
      }
    })

    expect(wrapper.text()).toContain('Longitudinal Statistical & Time-Control Consistency Analyzer')
    expect(wrapper.text()).toContain('Topic ID: fairplay')
  })

  it('filters topics by search query', async () => {
    const wrapper = mount(HelpModal)

    const searchInput = wrapper.find('input[placeholder="Search local manual..."]')
    await searchInput.setValue('Persona')

    expect(wrapper.text()).toContain('Player Personas & Aliases')
    expect(wrapper.text()).not.toContain('Scorebook, Notation & Publishing')
  })

  it('switches active topic when clicking on sidebar topic button', async () => {
    const wrapper = mount(HelpModal)

    const topicButtons = wrapper.findAll('.w-72 button')
    const personaBtn = topicButtons.find(b => b.text().includes('Player Personas'))
    expect(personaBtn).toBeDefined()
    await personaBtn!.trigger('click')

    expect(wrapper.text()).toContain('Multi-Alias Consolidation')
    expect(wrapper.text()).toContain('Incremental + Backfill Online Sync')
  })

  it('renders built-in System Man Page (man 1) viewer when tab is clicked', async () => {
    const wrapper = mount(HelpModal)

    const manBtn = wrapper.findAll('button').find(b => b.text().includes('System Man Page (man 1)'))
    expect(manBtn).toBeDefined()
    await manBtn!.trigger('click')

    expect(wrapper.text()).toContain('NEWSCID(1)')
    expect(wrapper.text()).toContain('newscid-import')
    expect(wrapper.text()).toContain('newscid-fairplay')
    expect(wrapper.text()).toContain('newscid-scout')
  })

  it('emits close when X button is clicked', async () => {
    const wrapper = mount(HelpModal)

    const closeBtn = wrapper.findAll('button').find(b => b.find('svg.lucide-x').exists() || b.html().includes('lucide-x'))
    expect(closeBtn).toBeDefined()
    await closeBtn!.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('renders tournaments & engine shootouts topic when selected', async () => {
    const wrapper = mount(HelpModal)

    const topicButtons = wrapper.findAll('.w-72 button')
    const tourneyBtn = topicButtons.find(b => b.text().includes('Tournaments & Engine Shootouts'))
    expect(tourneyBtn).toBeDefined()
    await tourneyBtn!.trigger('click')

    expect(wrapper.text()).toContain('Tournament Crosstables & Standings (F7)')
    expect(wrapper.text()).toContain('Sonneborn-Berger')
  })
})
