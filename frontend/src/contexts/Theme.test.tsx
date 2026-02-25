import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { ThemeProvider, useTheme } from './Theme'
import { createElement } from 'react'

function TestComponent() {
  const { theme, toggleTheme } = useTheme()
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <button onClick={toggleTheme}>Toggle</button>
    </div>
  )
}

describe('ThemeProvider', () => {
  let localStorageStore: Record<string, string> = {}

  const localStorageMock = {
    getItem: vi.fn((key: string) => localStorageStore[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      localStorageStore[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete localStorageStore[key]
    }),
    clear: vi.fn(() => {
      localStorageStore = {}
    }),
  }

  beforeEach(() => {
    localStorageStore = {}
    vi.clearAllMocks()
    document.documentElement.classList.remove('dark')
    
    Object.defineProperty(window, 'localStorage', { 
      value: localStorageMock,
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark')
  })

  it('should provide default light theme when no stored preference', () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme').textContent).toBe('light')
  })

  it('should use stored theme from localStorage', () => {
    localStorageStore['theme'] = 'dark'

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme').textContent).toBe('dark')
  })

  it('should toggle theme from light to dark', () => {
    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme').textContent).toBe('light')

    act(() => {
      fireEvent.click(screen.getByText('Toggle'))
    })

    expect(screen.getByTestId('theme').textContent).toBe('dark')
  })

  it('should toggle theme from dark to light', () => {
    localStorageStore['theme'] = 'dark'

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(screen.getByTestId('theme').textContent).toBe('dark')

    act(() => {
      fireEvent.click(screen.getByText('Toggle'))
    })

    expect(screen.getByTestId('theme').textContent).toBe('light')
  })

  it('should add dark class to document element when theme is dark', () => {
    localStorageStore['theme'] = 'dark'

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('should not have dark class when theme is light', () => {
    localStorageStore['theme'] = 'light'

    render(
      <ThemeProvider>
        <TestComponent />
      </ThemeProvider>
    )

    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('should throw error when useTheme is used outside ThemeProvider', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})

    expect(() => {
      render(createElement(TestComponent))
    }).toThrow('useTheme must be used within ThemeProvider')

    consoleError.mockRestore()
  })
})
