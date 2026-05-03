import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

export interface RobokassaSettingsValues {
  RobokassaEnabled: boolean
  RobokassaMerchantLogin: string
  RobokassaPassword1: string
  RobokassaPassword2: string
  RobokassaTestPassword1: string
  RobokassaTestPassword2: string
  RobokassaSandbox: boolean
  RobokassaCurrency: string
  RobokassaUnitPrice: number
  RobokassaMinTopUp: number
  RobokassaSignatureAlgo: string
  RobokassaResultUrl: string
  RobokassaSuccessUrl: string
  RobokassaFailUrl: string
}

interface Props {
  defaultValues: RobokassaSettingsValues
}

export function RobokassaSettingsSection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [loading, setLoading] = useState(false)

  const form = useForm<RobokassaSettingsValues>({
    defaultValues: props.defaultValues,
  })

  useEffect(() => {
    form.reset(props.defaultValues)
  }, [props.defaultValues, form])

  const handleSave = async () => {
    setLoading(true)
    try {
      const values = form.getValues()
      const options: { key: string; value: string }[] = [
        { key: 'RobokassaEnabled', value: String(values.RobokassaEnabled) },
        { key: 'RobokassaSandbox', value: String(values.RobokassaSandbox) },
        {
          key: 'RobokassaMerchantLogin',
          value: values.RobokassaMerchantLogin || '',
        },
        { key: 'RobokassaCurrency', value: values.RobokassaCurrency || 'RUB' },
        {
          key: 'RobokassaUnitPrice',
          value: String(values.RobokassaUnitPrice || 1),
        },
        {
          key: 'RobokassaMinTopUp',
          value: String(values.RobokassaMinTopUp || 1),
        },
        {
          key: 'RobokassaSignatureAlgo',
          value: values.RobokassaSignatureAlgo || 'md5',
        },
        { key: 'RobokassaResultUrl', value: values.RobokassaResultUrl || '' },
        { key: 'RobokassaSuccessUrl', value: values.RobokassaSuccessUrl || '' },
        { key: 'RobokassaFailUrl', value: values.RobokassaFailUrl || '' },
      ]
      if (values.RobokassaPassword1)
        options.push({
          key: 'RobokassaPassword1',
          value: values.RobokassaPassword1,
        })
      if (values.RobokassaPassword2)
        options.push({
          key: 'RobokassaPassword2',
          value: values.RobokassaPassword2,
        })
      if (values.RobokassaTestPassword1)
        options.push({
          key: 'RobokassaTestPassword1',
          value: values.RobokassaTestPassword1,
        })
      if (values.RobokassaTestPassword2)
        options.push({
          key: 'RobokassaTestPassword2',
          value: values.RobokassaTestPassword2,
        })

      for (const opt of options) {
        await updateOption.mutateAsync(opt)
      }
      toast.success(t('Updated successfully'))
    } catch {
      toast.error(t('Update failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <SettingsSection
      title={t('Robokassa Payment Gateway')}
      description={t(
        'Configure Robokassa payment integration for Russian and CIS users'
      )}
    >
      <Alert>
        <AlertDescription className='text-xs'>
          {t(
            'Configure ResultURL in Robokassa dashboard to: <ServerAddress>/api/robokassa/result. Use Password #2 for ResultURL signing.'
          )}
        </AlertDescription>
      </Alert>

      <div className='grid grid-cols-2 gap-4'>
        <div className='flex items-center gap-2'>
          <Switch
            checked={form.watch('RobokassaEnabled')}
            onCheckedChange={(v) => form.setValue('RobokassaEnabled', v)}
          />
          <Label>{t('Enable Robokassa')}</Label>
        </div>
        <div className='flex items-center gap-2'>
          <Switch
            checked={form.watch('RobokassaSandbox')}
            onCheckedChange={(v) => form.setValue('RobokassaSandbox', v)}
          />
          <Label>{t('Sandbox mode (IsTest=1)')}</Label>
        </div>
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('Merchant login')}</Label>
        <Input {...form.register('RobokassaMerchantLogin')} />
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Password #1 (Production)')}</Label>
          <Input
            type='password'
            placeholder={t('Leave blank to keep existing')}
            {...form.register('RobokassaPassword1')}
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Password #2 (Production)')}</Label>
          <Input
            type='password'
            placeholder={t('Leave blank to keep existing')}
            {...form.register('RobokassaPassword2')}
          />
        </div>
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Password #1 (Sandbox)')}</Label>
          <Input
            type='password'
            placeholder={t('Leave blank to keep existing')}
            {...form.register('RobokassaTestPassword1')}
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Password #2 (Sandbox)')}</Label>
          <Input
            type='password'
            placeholder={t('Leave blank to keep existing')}
            {...form.register('RobokassaTestPassword2')}
          />
        </div>
      </div>

      <div className='grid grid-cols-3 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Currency')}</Label>
          <Select
            value={form.watch('RobokassaCurrency') || 'RUB'}
            onValueChange={(v) => form.setValue('RobokassaCurrency', v)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='RUB'>RUB</SelectItem>
              <SelectItem value='USD'>USD</SelectItem>
              <SelectItem value='EUR'>EUR</SelectItem>
              <SelectItem value='KZT'>KZT</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Unit price')}</Label>
          <Input
            type='number'
            step={0.01}
            min={0}
            {...form.register('RobokassaUnitPrice', { valueAsNumber: true })}
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Minimum top-up quantity')}</Label>
          <Input
            type='number'
            min={1}
            {...form.register('RobokassaMinTopUp', { valueAsNumber: true })}
          />
        </div>
      </div>

      <div className='grid grid-cols-3 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Signature algorithm')}</Label>
          <Select
            value={form.watch('RobokassaSignatureAlgo') || 'md5'}
            onValueChange={(v) => form.setValue('RobokassaSignatureAlgo', v)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='md5'>MD5</SelectItem>
              <SelectItem value='sha256'>SHA-256</SelectItem>
              <SelectItem value='sha512'>SHA-512</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Success URL (optional)')}</Label>
          <Input {...form.register('RobokassaSuccessUrl')} />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Fail URL (optional)')}</Label>
          <Input {...form.register('RobokassaFailUrl')} />
        </div>
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('ResultURL (for reference, configure in Robokassa)')}</Label>
        <Input
          placeholder='https://example.com/api/robokassa/result'
          {...form.register('RobokassaResultUrl')}
        />
      </div>

      <Button onClick={handleSave} disabled={loading}>
        {loading ? t('Saving...') : t('Save Changes')}
      </Button>
    </SettingsSection>
  )
}
