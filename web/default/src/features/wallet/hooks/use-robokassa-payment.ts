import { useState, useCallback } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import { requestRobokassaPayment, isApiSuccess } from '../api'
import { submitPaymentForm } from '../lib'

function getErrorMessage(message: string | undefined, data: unknown): string {
  if (typeof data === 'string' && data.trim()) {
    return data
  }
  return message || i18next.t('Payment request failed')
}

/**
 * Hook for handling Robokassa payment processing.
 *
 * Robokassa expects the user's browser to POST a signed form to its hosted
 * checkout page (auth.robokassa.ru). The backend returns the form fields and
 * the target URL; we reuse `submitPaymentForm` to perform the redirect.
 */
export function useRobokassaPayment() {
  const [processing, setProcessing] = useState(false)

  const processRobokassaPayment = useCallback(async (topupAmount: number) => {
    setProcessing(true)
    try {
      const response = await requestRobokassaPayment({
        amount: Math.floor(topupAmount),
      })

      if (
        isApiSuccess(response) &&
        response.url &&
        response.data &&
        typeof response.data === 'object'
      ) {
        submitPaymentForm(
          response.url,
          response.data as Record<string, unknown>
        )
        toast.success(i18next.t('Redirecting to payment page...'))
        return true
      }

      toast.error(getErrorMessage(response.message, response.data))
      return false
    } catch (_error) {
      toast.error(i18next.t('Payment request failed'))
      return false
    } finally {
      setProcessing(false)
    }
  }, [])

  return { processing, processRobokassaPayment }
}
