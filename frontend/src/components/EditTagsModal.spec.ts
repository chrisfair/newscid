import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EditTagsModal from './EditTagsModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  UpdateGameHeaders: vi.fn().mockImplementation((gameId: number, tags: Record<string, string>) => {
    return Promise.resolve({
      id: gameId,
      white: tags.White || 'White',
      black: tags.Black || 'Black',
      whiteElo: parseInt(tags.WhiteElo) || 0,
      blackElo: parseInt(tags.BlackElo) || 0,
      event: tags.Event || 'Event',
      site: tags.Site || 'Site',
      date: tags.Date || '2020.01.01',
      round: tags.Round || '1',
      result: tags.Result || '*',
      eco: tags.ECO || 'C50',
      moves: 20,
      deleted: false
    })
  })
}))

describe('EditTagsModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal with initial tag inputs', async () => {
    const wrapper = mount(EditTagsModal, {
      props: {
        gameId: 42,
        initialTags: {
          White: 'Kasparov, Garry',
          Black: 'Karpov, Anatoly',
          WhiteElo: '2800',
          BlackElo: '2750',
          Event: 'World Championship',
          Site: 'Moscow',
          Date: '1985.11.09',
          Round: '24',
          Result: '1-0',
          ECO: 'B44'
        }
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Edit Game Headers & Tags')
    expect(wrapper.text()).toContain('Game #42')

    const whiteInput = wrapper.find('input[placeholder*="Kasparov"]')
    expect(whiteInput.exists()).toBe(true)
    expect((whiteInput.element as HTMLInputElement).value).toBe('Kasparov, Garry')
  })

  it('submits updated headers and emits tagsUpdated', async () => {
    const wrapper = mount(EditTagsModal, {
      props: {
        gameId: 5,
        initialTags: {
          White: 'Player 1',
          Black: 'Player 2',
          Result: '*'
        }
      }
    })
    await flushPromises()

    const whiteInput = wrapper.find('input[placeholder*="Kasparov"]')
    await whiteInput.setValue('Carlsen, Magnus')

    const whiteEloInput = wrapper.find('input[placeholder*="2800"]')
    await whiteEloInput.setValue('2882')

    // Select Result 1-0
    const resBtn = wrapper.findAll('button[type="button"]').find(b => b.text().includes('1-0'))
    expect(resBtn).toBeDefined()
    await resBtn!.trigger('click')

    const submitBtn = wrapper.find('button[type="submit"]')
    await submitBtn.trigger('click')
    await flushPromises()

    expect(AppApi.UpdateGameHeaders).toHaveBeenCalledWith(5, expect.objectContaining({
      White: 'Carlsen, Magnus',
      WhiteElo: '2882',
      Result: '1-0'
    }))

    expect(wrapper.emitted('tagsUpdated')).toBeTruthy()
  })
})
