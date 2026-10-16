import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SupportModal from './SupportModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetSupportStatus: vi.fn(),
  RecordUserSupport: vi.fn(),
  DismissSupportNag: vi.fn(),
  OpenSupportLink: vi.fn(),
}))

describe('SupportModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppBindings.GetSupportStatus).mockResolvedValue({
      supported: false,
      isAuthor: false,
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
    expect(AppBindings.OpenSupportLink).toHaveBeenCalledWith(10)
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
