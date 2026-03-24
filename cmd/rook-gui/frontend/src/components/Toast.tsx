import { useEffect } from 'react'

interface ToastProps {
  message: string
  type?: 'success' | 'error' | 'info'
  onClose: () => void
}

export function Toast({ message, type = 'info', onClose }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  const bgColor = type === 'error' ? 'bg-rook-error' : type === 'success' ? 'bg-rook-success' : 'bg-rook-active'

  return (
    <div className={`fixed bottom-4 right-4 ${bgColor} text-white px-4 py-2 shadow-lg text-xs z-50`}>
      {message}
    </div>
  )
}
