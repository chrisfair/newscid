import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TWICDownloadModal from './TWICDownloadModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetLatestTWICIssue: vi.fn().mockResolvedValue(1560),
  CheckTWICIssue: vi.fn().mockImplementation((issue: number) => {
    return Promise.resolve(issue <= 1560)
  }),
  DownloadTWICIssue: vi.fn().mockImplementation((issue: number) => {
    return Promise.resolve({
      issueNumber: issue,
      url: `https://theweekinchess.com/zips/twic${issue}g.zip`,
      gamesDownloaded: 2500,
      gamesImported: 2500,
      durationMs: 450
    })
  }),
  DownloadTWICRange: vi.fn().mockImplementation((start: number, end: number) => {
    return Promise.resolve({
      startIssue: start,
      endIssue: end,
      totalDownloaded: 5000,
      totalImported: 5000,
      durationMs: 900,
      issues: [
        { issueNumber: start, gamesDownloaded: 2500, gamesImported: 2500, durationMs: 400 },
        { issueNumber: end, gamesDownloaded: 2500, gamesImported: 2500, durationMs: 450 }
      ]
    })
  })
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn()
}))

describe('TWICDownloadModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal, calls GetLatestTWICIssue on mount and sets latest issue', async () => {
    const wrapper = mount(TWICDownloadModal, {
      props: { dbName: 'MasterBase.si4' }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('The Week In Chess (TWIC) Downloader')
    expect(wrapper.text()).toContain('MasterBase.si4')
    expect(AppApi.GetLatestTWICIssue).toHaveBeenCalled()

    const input = wrapper.find('input[type="number"]')
    expect(input.exists()).toBe(true)
    expect((input.element as HTMLInputElement).value).toBe('1560')
  })

  it('verifies issue online when Verify Online button is clicked', async () => {
    const wrapper = mount(TWICDownloadModal)
    await flushPromises()

    const verifyBtn = wrapper.findAll('button').find(b => b.text().includes('Verify Online'))
    expect(verifyBtn).toBeDefined()
    await verifyBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.CheckTWICIssue).toHaveBeenCalledWith(1560)
    expect(wrapper.text()).toContain('verified and ready on the server')
  })

  it('downloads single TWIC issue and emits imported', async () => {
    const wrapper = mount(TWICDownloadModal)
    await flushPromises()

    const downloadBtn = wrapper.findAll('button').find(b => b.text().includes('Download & Import'))
    expect(downloadBtn).toBeDefined()
    await downloadBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.DownloadTWICIssue).toHaveBeenCalledWith(1560)
    expect(wrapper.text()).toContain('Successfully downloaded & imported 2,500 games from TWIC #1560')
    expect(wrapper.emitted('imported')).toBeTruthy()
    expect(wrapper.emitted('imported')![0]).toEqual([2500])
  })

  it('switches to range mode and downloads range of issues', async () => {
    const wrapper = mount(TWICDownloadModal)
    await flushPromises()

    const rangeTabBtn = wrapper.findAll('button').find(b => b.text().includes('Issue Range / Batch'))
    expect(rangeTabBtn).toBeDefined()
    await rangeTabBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Start Issue')
    expect(wrapper.text()).toContain('End Issue')

    const downloadBtn = wrapper.findAll('button').find(b => b.text().includes('Download & Import'))
    expect(downloadBtn).toBeDefined()
    await downloadBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.DownloadTWICRange).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Successfully downloaded & imported 5,000 games')
    expect(wrapper.emitted('imported')).toBeTruthy()
    expect(wrapper.emitted('imported')![0]).toEqual([5000])
  })
})
