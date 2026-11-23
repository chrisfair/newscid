import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SupportModal from './SupportModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetSupportStatus: vi.fn(),
  RecordUserSupport: vi.fn(),
  DismissSupportNag: vi.fn(),
  OpenSupportLink: vi.fn(),
  SyncRemoteSupportStatus: vi.fn(),
  PurchaseMicrosoftStoreUnlock: vi.fn(),
}))

describe('SupportModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppBindings.GetSupportStatus).mockResolvedValue({
      supported: false,
      isAuthor: false,
      isPackagedApp: false,
      isStorePurchase: false,
      authorId: '',
      deviceSignature: '',
      supporterTier: '',
      supportedAt: 0,
      dismissCount: 0,
      lastDismissedAt: 0,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: true,
    } as any)
    vi.mocked(AppBindings.OpenSupportLink).mockResolvedValue('https://donate.stripe.com/newscid-support')
    vi.mocked(AppBindings.DismissSupportNag).mockResolvedValue({
      supported: false,
      isAuthor: false,
      dismissCount: 1,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: true,
    } as any)
    vi.mocked(AppBindings.RecordUserSupport).mockResolvedValue({
      supported: true,
      isAuthor: false,
      supporterTier: '$10 Support',
      supportedAt: Date.now(),
      dismissCount: 0,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: false,
    } as any)
    vi.mocked(AppBindings.SyncRemoteSupportStatus).mockResolvedValue({
      supported: true,
      isAuthor: false,
      supporterHandle: 'HeroPlayer',
      supporterTier: '$35 Support',
      dismissCount: 0,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: false,
    } as any)
    vi.mocked(AppBindings.PurchaseMicrosoftStoreUnlock).mockResolvedValue({
      purchased: true,
      message: 'Microsoft Store $9.99 Permanent Unlock activated!',
    } as any)
  })

  it('renders support tiers ($5 minimum) and opens Stripe link on click', async () => {
    const wrapper = mount(SupportModal)
    await flushPromises()

    expect(wrapper.text()).toContain('Support NewScid')
    expect(wrapper.text()).toContain('$5 or higher')

    // Default is $5
    const stripeBtn = wrapper.get('[data-testid="stripe-support-btn"]')
    expect(stripeBtn.text()).toContain('Support via Stripe ($5+)')

    // Switch to $10 tier
    await wrapper.get('[data-testid="tier-10"]').trigger('click')
    expect(stripeBtn.text()).toContain('Support via Stripe ($10+)')

    // Click Stripe button
    await stripeBtn.trigger('click')
    await flushPromises()
    expect(AppBindings.OpenSupportLink).toHaveBeenCalledWith(10)
    expect(wrapper.emitted('updated')).toBeTruthy()
  })

  it('falls back to window.open when OpenSupportLink rejects', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    vi.mocked(AppBindings.OpenSupportLink).mockRejectedValueOnce(new Error('wails unavailable'))

    const wrapper = mount(SupportModal)
    await flushPromises()

    await wrapper.get('[data-testid="stripe-support-btn"]').trigger('click')
    await flushPromises()

    expect(openSpy).toHaveBeenCalledWith('https://donate.stripe.com/newscid-support', '_blank', 'noopener,noreferrer')
    openSpy.mockRestore()
  })

  it('enforces $5 minimum on custom support amount', async () => {
    const wrapper = mount(SupportModal)
    await flushPromises()

    await wrapper.get('[data-testid="tier-custom"]').trigger('click')
    const input = wrapper.get('[data-testid="custom-amount-input"]')
    await input.setValue(3) // Below $5 minimum -> clamped to $5
    expect(wrapper.get('[data-testid="stripe-support-btn"]').text()).toContain('Support via Stripe ($5+)')

    await input.setValue(15)
    expect(wrapper.get('[data-testid="stripe-support-btn"]').text()).toContain('Support via Stripe ($15+)')
  })

  it('records user support and disables future startup nag screen', async () => {
    const wrapper = mount(SupportModal)
    await flushPromises()

    await wrapper.get('[data-testid="support-reference-input"]').setValue('rcpt_stripe_999')
    await wrapper.get('[data-testid="confirm-support-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.RecordUserSupport).toHaveBeenCalledWith('$5 Support', 'rcpt_stripe_999')
    expect(wrapper.find('[data-testid="supporter-active-banner"]').exists()).toBe(true)
    expect(wrapper.emitted('updated')).toBeTruthy()
  })

  it('renders Microsoft Store $9.99 permanent unlock flow on Free Store edition and activates license', async () => {
    vi.mocked(AppBindings.GetSupportStatus)
      .mockResolvedValueOnce({
        supported: false,
        isAuthor: false,
        isWindows: true,
        isPackagedApp: true,
        isStorePurchase: false,
        canPurchaseInApp: true,
        storePriceUsd: 9.99,
        dismissCount: 0,
        stripeUrl: 'https://donate.stripe.com/newscid-support',
        minAmountUsd: 5,
        shouldShowNag: true,
      } as any)
      .mockResolvedValueOnce({
        supported: true,
        isLifetime: true,
        isAuthor: false,
        isWindows: true,
        isPackagedApp: true,
        isStorePurchase: true,
        dismissCount: 0,
        stripeUrl: 'https://donate.stripe.com/newscid-support',
        minAmountUsd: 5,
        shouldShowNag: false,
      } as any)

    const wrapper = mount(SupportModal)
    await flushPromises()

    // Packaged Store app shows MS Store button instead of Stripe button
    expect(wrapper.find('[data-testid="stripe-support-btn"]').exists()).toBe(false)
    const storeBtn = wrapper.get('[data-testid="msstore-purchase-btn"]')
    expect(storeBtn.text()).toContain('$9.99 (Microsoft Store)')

    await storeBtn.trigger('click')
    await flushPromises()

    expect(AppBindings.PurchaseMicrosoftStoreUnlock).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="store-verified-banner"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="confirm-support-msg"]').text()).toContain('Permanent Unlock activated')
    expect(wrapper.emitted('updated')?.[0]).toEqual([true])
  })

  it('handles cancelled and errored Microsoft Store purchase attempts gracefully', async () => {
    vi.mocked(AppBindings.GetSupportStatus).mockResolvedValue({
      supported: false,
      isAuthor: false,
      isPackagedApp: true,
      isStorePurchase: false,
      dismissCount: 0,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: true,
    } as any)
    vi.mocked(AppBindings.PurchaseMicrosoftStoreUnlock)
      .mockResolvedValueOnce({ purchased: false, message: 'Purchase cancelled by user.' } as any)
      .mockRejectedValueOnce(new Error('Store IPC error'))

    const wrapper = mount(SupportModal)
    await flushPromises()

    await wrapper.get('[data-testid="msstore-purchase-btn"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="confirm-support-msg"]').text()).toContain('Purchase cancelled by user.')

    await wrapper.get('[data-testid="msstore-purchase-btn"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="confirm-support-msg"]').text()).toContain('Microsoft Store checkout could not be opened.')
  })

  it('restores supporter status via SyncRemoteSupportStatus button and handles not-found/error states', async () => {
    const wrapper = mount(SupportModal)
    await flushPromises()

    const syncBtn = wrapper.get('[data-testid="sync-shared-support-btn"]')
    expect(syncBtn.attributes('disabled')).toBeDefined()

    // Case 1: Not found
    vi.mocked(AppBindings.SyncRemoteSupportStatus).mockResolvedValueOnce({
      supported: false,
      isAuthor: false,
      dismissCount: 0,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: true,
    } as any)
    await wrapper.get('[data-testid="support-reference-input"]').setValue('UnknownHandle')
    await syncBtn.trigger('click')
    await flushPromises()
    expect(AppBindings.SyncRemoteSupportStatus).toHaveBeenCalledWith('UnknownHandle')
    expect(wrapper.get('[data-testid="confirm-support-msg"]').text()).toContain('No prior contribution record found')

    // Case 2: Network error
    vi.mocked(AppBindings.SyncRemoteSupportStatus).mockRejectedValueOnce(new Error('offline'))
    await syncBtn.trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="confirm-support-msg"]').text()).toContain('Could not reach the shared supporter database')

    // Case 3: Restored successfully
    await wrapper.get('[data-testid="support-reference-input"]').setValue('HeroPlayer')
    await syncBtn.trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="confirm-support-msg"]').text()).toContain('Supporter status restored for HeroPlayer')
    expect(wrapper.find('[data-testid="supporter-active-banner"]').exists()).toBe(true)
    expect(wrapper.emitted('updated')).toBeTruthy()
  })

  it('displays author verified banner when Chess.com and Lichess IDs match author machine', async () => {
    vi.mocked(AppBindings.GetSupportStatus).mockResolvedValueOnce({
      supported: true,
      isAuthor: true,
      authorId: 'chess.com:ChristopherFair|lichess.org:QuenyenHighElf',
      deviceSignature: 'a1b2c3d4e5f6',
      supporterTier: 'Author',
      supportedAt: Date.now(),
      dismissCount: 0,
      stripeUrl: 'https://donate.stripe.com/newscid-support',
      minAmountUsd: 5,
      shouldShowNag: false,
    } as any)

    const wrapper = mount(SupportModal)
    await flushPromises()

    const banner = wrapper.find('[data-testid="author-verified-banner"]')
    expect(banner.exists()).toBe(true)
    expect(banner.text()).toContain('ChristopherFair')
    expect(banner.text()).toContain('QuenyenHighElf')
  })

  it('dismisses non-obligatory prompt and emits close when Maybe Later is clicked', async () => {
    const wrapper = mount(SupportModal)
    await flushPromises()

    await wrapper.get('[data-testid="dismiss-support-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.DismissSupportNag).toHaveBeenCalled()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
