import { Button } from '../components/tremor/Button'
import { useState } from 'react'
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from '@tanstack/react-router'
import { useToast } from '../hooks/useToast'
import { translate } from '../lib/translate'
import { useQueryApi } from '../hooks/useQueryApi'

export const Route = createFileRoute('/admin/security')({
  component: Admin,
})

function Admin() {
  const [isResetting, setIsResetting] = useState(false)
  const { toast } = useToast()
  const queryApi = useQueryApi()
  const navigate = useNavigate({ from: '/admin' })

  const handleReset = async () => {
    setIsResetting(true)
    try {
      await queryApi('/api/admin/reset-jwt-secret', {
        method: 'POST',
      })
      toast({
        title: translate('Success'),
        description: translate('JWT secret reset successfully'),
      })
    } catch (error) {
      if (isRedirect(error)) {
        return navigate(error)
      }
      toast({
        title: translate('Error'),
        description:
          error instanceof Error
            ? error.message
            : translate('An error occurred'),
        variant: 'error',
      })
    } finally {
      setIsResetting(false)
    }
  }

  return (
    <div>
      <h2 className="text-xl font-semibold mb-4">
        {translate('Security Settings')}
      </h2>
      <div className="space-y-4">
        <div>
          <h3 className="text-lg font-medium mb-2">JWT Secret</h3>
          <p className="text-gray-600 dark:text-gray-400 mb-4">
            {translate(
              'Reset the JWT secret to invalidate all existing tokens.',
            )}
          </p>
          <Button onClick={handleReset} disabled={isResetting} variant="secondary">
            {isResetting ? translate('Resetting...') : translate('Reset JWT Secret')}
          </Button>
        </div>
      </div>
    </div>
  )
}
