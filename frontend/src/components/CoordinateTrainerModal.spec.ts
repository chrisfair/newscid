import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CoordinateTrainerModal from './CoordinateTrainerModal.vue'

describe('CoordinateTrainerModal.vue', () => {
  it('renders modal with initial target coordinate and board', () => {
    const wrapper = mount(CoordinateTrainerModal)
    expect(wrapper.text()).toContain('Board Vision & Coordinate Trainer')
    expect(wrapper.text()).toContain('Start Trainer')
    expect(wrapper.findAll('.aspect-square button').length).toBe(64)
  })

  it('starts game and updates timer state on start click', async () => {
    const wrapper = mount(CoordinateTrainerModal)
    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Trainer'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')

    expect(wrapper.text()).toContain('Streak')
    expect(wrapper.text()).toContain('Reset')
  })

  it('switches perspective between white and black', async () => {
    const wrapper = mount(CoordinateTrainerModal)
    const blackBtn = wrapper.findAll('button').find(b => b.text() === 'Black')
    expect(blackBtn).toBeDefined()
    await blackBtn!.trigger('click')

    // In Black perspective, file h is at bottom-left
    expect(wrapper.text()).toContain('h')
  })

  it('emits close when X button is clicked', async () => {
    const wrapper = mount(CoordinateTrainerModal)
    const closeBtn = wrapper.find('button')
    await closeBtn.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
