import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Chess960Modal from './Chess960Modal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetChess960Position: vi.fn().mockImplementation((id: number) => {
    return Promise.resolve({
      id,
      pieces: id === 518 ? 'RNBQKBNR' : 'BBQNNRKR',
      fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
      isClassic: id === 518,
    })
  }),
  GenerateRandomChess960: vi.fn().mockResolvedValue({
    id: 42,
    pieces: 'BBNNQRKR',
    fen: 'bbnnqrkr/pppppppp/8/8/8/8/PPPPPPPP/BBNNQRKR w KQkq - 0 1',
    isClassic: false,
  }),
}))

describe('Chess960Modal.vue', () => {
  it('renders modal with title and fetches initial classic position 518', async () => {
    const wrapper = mount(Chess960Modal)
    expect(wrapper.text()).toContain('Chess960 (Fischer Random)')
    await new Promise(r => setTimeout(r, 10))
    expect(wrapper.text()).toContain('Standard Classical Layout')
    expect(wrapper.text()).toContain('RNBQKBNR')
  })

  it('rolls random position when Random button is clicked', async () => {
    const wrapper = mount(Chess960Modal)
    await new Promise(r => setTimeout(r, 10))

    const randomBtn = wrapper.findAll('button').find(b => b.text().includes('Random'))
    expect(randomBtn).toBeDefined()
    await randomBtn!.trigger('click')
    await new Promise(r => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Position #42')
  })

  it('emits loadPosition and close when Load Into Board is clicked', async () => {
    const wrapper = mount(Chess960Modal)
    await new Promise(r => setTimeout(r, 10))

    const loadBtn = wrapper.findAll('button').find(b => b.text().includes('Load Into Board'))
    expect(loadBtn).toBeDefined()
    await loadBtn!.trigger('click')

    expect(wrapper.emitted('loadPosition')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
