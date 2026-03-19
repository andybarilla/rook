import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import { Toast } from '../components/Toast'

interface ToastState {
  message: string
  type: 'success' | 'error' | 'info'
}

interface ToastContextType {
  show: (message: string, type?: 'success' | 'error' | 'info') => void
}

const ToastContext = createContext<ToastContextType | null>(null)

interface ToastProviderProps {
  children: ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toast, setToast] = useState<ToastState | null>(null)

  const show = useCallback((message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ message, type })
  }, [])

  const hide = useCallback(() => {
    setToast(null)
  }, [])

  return (
    <ToastContext.Provider value={{ show }}>
      {children}
      {toast && <Toast message={toast.message} type={toast.type} onClose={hide} />}
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextType {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}
