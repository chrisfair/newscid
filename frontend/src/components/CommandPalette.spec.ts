import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CommandPalette from './CommandPalette.vue'
import { FileText, Cpu } from '@lucide/vue'

describe('CommandPalette.vue', () => {
  const dummyAction = vi.fn()
  const commands = [
    {
      id: 'cmd-new',
      title: 'New Game',
      subtitle: 'Start a blank chess game',
      category: 'Actions' as const,
      icon: FileText,
      shortcut: 'Ctrl+N',
      action: dummyAction,
    },
    {
      id: 'cmd-engine',
      title: 'Start Live Analysis',
      subtitle: 'Toggle engine evaluation',
      category: 'Analysis' as const,
      icon: Cpu,
      shortcut: 'F2',
      action: vi.fn(),
    },
  ]

  it('renders modal when isOpen is true', () => {
    const wrapper = mount(CommandPalette, {
      props: {
        isOpen: true,
        commands,
      },
    })
    expect(wrapper.find('input').exists()).toBe(true)
    expect(wrapper.text()).toContain('New Game')
    expect(wrapper.text()).toContain('Start Live Analysis')
  })

  it('filters commands when searching', async () => {
    const wrapper = mount(CommandPalette, {
      props: {
        isOpen: true,
        commands,
      },
    })
    const input = wrapper.find('input')
    await input.setValue('live')
    expect(wrapper.text()).toContain('Start Live Analysis')
    expect(wrapper.text()).not.toContain('New Game')
  })

  it('executes command action on click', async () => {
    const wrapper = mount(CommandPalette, {
      props: {
        isOpen: true,
        commands,
      },
    })
    const firstCmd = wrapper.findAll('.cursor-pointer').find((el) => el.text().includes('New Game'))
    expect(firstCmd).toBeDefined()
    await firstCmd!.trigger('click')
    expect(dummyAction).toHaveBeenCalled()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
