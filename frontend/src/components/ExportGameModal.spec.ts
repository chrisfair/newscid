import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ExportGameModal from './ExportGameModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  ExportCurrentGameHTML: vi.fn(),
  ExportCurrentGameLaTeX: vi.fn(),
}))

describe('ExportGameModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal when open and fetches HTML export by default', async () => {
    vi.mocked(AppBindings.ExportCurrentGameHTML).mockResolvedValue('<!DOCTYPE html><html><body>Carlsen vs Nakamura</body></html>')

    const wrapper = mount(ExportGameModal, {
      props: {
        isOpen: true,
      },
    })

    expect(wrapper.text()).toContain('Export Game (HTML & LaTeX)')
    expect(wrapper.text()).toContain('Modern HTML')

    // Wait for DOM update after async call
    await vi.waitFor(() => {
      expect(AppBindings.ExportCurrentGameHTML).toHaveBeenCalled()
    })
    await vi.waitFor(() => {
      expect(wrapper.find('iframe').exists()).toBe(true)
    })

    expect(wrapper.find('iframe').attributes('srcdoc')).toContain('Carlsen vs Nakamura')
  })

  it('switches to LaTeX format and fetches LaTeX export', async () => {
    vi.mocked(AppBindings.ExportCurrentGameHTML).mockResolvedValue('<html>HTML</html>')
    vi.mocked(AppBindings.ExportCurrentGameLaTeX).mockResolvedValue('\\documentclass{article}\n\\begin{document}LaTeX Game\\end{document}')

    const wrapper = mount(ExportGameModal, {
      props: {
        isOpen: true,
      },
    })

    await vi.waitFor(() => {
      expect(wrapper.find('iframe').exists()).toBe(true)
    })

    // Click LaTeX format button
    const latexBtn = wrapper.findAll('button').find(b => b.text().includes('LaTeX Article'))
    expect(latexBtn).toBeDefined()
    await latexBtn?.trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('\\documentclass{article}')
    })
  })

  it('switches to Source Code tab and shows code in textarea', async () => {
    vi.mocked(AppBindings.ExportCurrentGameHTML).mockResolvedValue('<!DOCTYPE html><code>test</code>')

    const wrapper = mount(ExportGameModal, {
      props: {
        isOpen: true,
      },
    })

    await vi.waitFor(() => {
      expect(AppBindings.ExportCurrentGameHTML).toHaveBeenCalled()
    })

    // Click Source Code tab
    const codeTab = wrapper.findAll('button').find(b => b.text() === 'Source Code')
    expect(codeTab).toBeDefined()
    await codeTab?.trigger('click')

    const textarea = wrapper.find('textarea')
    expect(textarea.exists()).toBe(true)
    expect((textarea.element as HTMLTextAreaElement).value).toBe('<!DOCTYPE html><code>test</code>')
  })

  it('displays error message if export call rejects', async () => {
    vi.mocked(AppBindings.ExportCurrentGameHTML).mockRejectedValue(new Error('No active game loaded'))

    const wrapper = mount(ExportGameModal, {
      props: {
        isOpen: true,
      },
    })

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Failed to export game')
      expect(wrapper.text()).toContain('No active game loaded')
    })
  })

  it('emits close event when close button is clicked', async () => {
    const wrapper = mount(ExportGameModal, {
      props: {
        isOpen: true,
      },
    })

    const closeBtn = wrapper.find('button[title="Close (Esc)"]')
    await closeBtn.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
