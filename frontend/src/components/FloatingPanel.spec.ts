import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import FloatingPanel from './FloatingPanel.vue'
import { workspacePanels, resetWorkspaceLayout, togglePanelFloater } from '../utils/workspaceLayout'

describe('FloatingPanel.vue', () => {
  beforeEach(() => {
    resetWorkspaceLayout()
    togglePanelFloater('engine')
  })

  it('supports dragging the floater header and emits focus + updateRect', async () => {
    const wrapper = mount(FloatingPanel, {
      props: { panel: workspacePanels.value.engine },
      slots: { default: '<div>Engine Content</div>' },
    })

    const header = wrapper.get('.floater-header')
    await header.trigger('mousedown', { clientX: 100, clientY: 100 })

    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 160, clientY: 140 }))
    window.dispatchEvent(new MouseEvent('mouseup'))

    expect(wrapper.emitted('focus')).toBeTruthy()
    const updates = wrapper.emitted('updateRect')
    expect(updates).toBeTruthy()
    expect(updates![0][0]).toBe('engine')
    expect(updates![0][1]).toMatchObject({ x: expect.any(Number), y: expect.any(Number) })
  })

  it('supports resizing via bottom-right handle and emits updateRect', async () => {
    const wrapper = mount(FloatingPanel, {
      props: { panel: workspacePanels.value.engine },
    })

    const resizer = wrapper.get('.floater-resizer')
    await resizer.trigger('mousedown', { clientX: 500, clientY: 400 })

    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 580, clientY: 460 }))
    window.dispatchEvent(new MouseEvent('mouseup'))

    const updates = wrapper.emitted('updateRect')
    expect(updates).toBeTruthy()
    expect(updates![0][1]).toMatchObject({ width: expect.any(Number), height: expect.any(Number) })
  })

  it('handles minimize toggle, dock, detachScreen, workspace select, and hide actions', async () => {
    const wrapper = mount(FloatingPanel, {
      props: { panel: workspacePanels.value.engine },
    })

    // Double-click header toggles minimize
    await wrapper.get('.floater-header').trigger('dblclick')
    expect(wrapper.emitted('updateRect')?.[0]).toEqual(['engine', { minimized: true }])

    // Screen button emits detachScreen
    const screenBtn = wrapper.findAll('button').find(b => b.text().includes('Screen'))
    await screenBtn!.trigger('click')
    expect(wrapper.emitted('detachScreen')?.[0]).toEqual(['engine', ''])

    // Workspace dropdown emits detachScreen with target workspace
    const select = wrapper.get('select')
    await select.setValue('4')
    expect(wrapper.emitted('detachScreen')?.[1]).toEqual(['engine', '4'])

    // Hide button emits hide
    const hideBtn = wrapper.get('button[title="Hide / remove panel"]')
    await hideBtn.trigger('click')
    expect(wrapper.emitted('hide')?.[0]).toEqual(['engine'])
  })
})
