import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { AuthProvider, useAuth } from './Auth'
import * as apiModule from '../lib/api'
import { createElement } from 'react'

function TestComponent() {
  const { user, loading, login, register, logout } = useAuth()
  return (
    <div>
      <span data-testid="loading">{loading.toString()}</span>
      <span data-testid="user">{user ? user.username : 'null'}</span>
      <button onClick={() => login('testuser', 'Password123')}>Login</button>
      <button onClick={() => register('newuser', 'Password123')}>Register</button>
      <button onClick={logout}>Logout</button>
    </div>
  )
}

const fakeTokenResponse = {
  access_token: 'fake-access-token',
  refresh_token: 'fake-refresh-token',
  expires_in: 900,
  token_type: 'Bearer',
}

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('should show loading state initially', () => {
    localStorage.setItem('nuistagram_access_token', 'fake-token')
    vi.spyOn(apiModule.api, 'getMe').mockImplementation(() => new Promise(() => {}))

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    expect(screen.getByTestId('loading').textContent).toBe('true')
  })

  it('should set user when getMe succeeds', async () => {
    localStorage.setItem('nuistagram_access_token', 'fake-token')
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'testuser' })

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('loading').textContent).toBe('false')
    })

    expect(screen.getByTestId('user').textContent).toBe('testuser')
  })

  it('should set user to null when getMe fails', async () => {
    localStorage.setItem('nuistagram_access_token', 'fake-token')
    vi.spyOn(apiModule.api, 'getMe').mockRejectedValue(new Error('Unauthorized'))

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('loading').textContent).toBe('false')
    })

    expect(screen.getByTestId('user').textContent).toBe('null')
  })

  it('should login and set user', async () => {
    // No token → init skips getMe; getMe is only called after login
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'testuser' })
    vi.spyOn(apiModule.api, 'login').mockResolvedValue(fakeTokenResponse)

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('loading').textContent).toBe('false')
    })

    await act(async () => {
      screen.getByText('Login').click()
    })

    await waitFor(() => {
      expect(screen.getByTestId('user').textContent).toBe('testuser')
    })

    expect(apiModule.api.login).toHaveBeenCalledWith('testuser', 'Password123')
  })

  it('should register and set user', async () => {
    // No token → init skips getMe; getMe is only called after register
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'newuser' })
    vi.spyOn(apiModule.api, 'register').mockResolvedValue(fakeTokenResponse)

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('loading').textContent).toBe('false')
    })

    await act(async () => {
      screen.getByText('Register').click()
    })

    await waitFor(() => {
      expect(screen.getByTestId('user').textContent).toBe('newuser')
    })

    expect(apiModule.api.register).toHaveBeenCalledWith('newuser', 'Password123')
  })

  it('should logout and clear user', async () => {
    // Set token so init calls getMe and populates user
    localStorage.setItem('nuistagram_access_token', 'fake-token')
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'testuser' })
    vi.spyOn(apiModule.api, 'logout').mockResolvedValue({ success: true })

    render(
      <AuthProvider>
        <TestComponent />
      </AuthProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('user').textContent).toBe('testuser')
    })

    await act(async () => {
      screen.getByText('Logout').click()
    })

    await waitFor(() => {
      expect(screen.getByTestId('user').textContent).toBe('null')
    })

    expect(apiModule.api.logout).toHaveBeenCalled()
  })

  it('should throw error when useAuth is used outside AuthProvider', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})

    expect(() => {
      render(createElement(TestComponent))
    }).toThrow('useAuth must be used within AuthProvider')

    consoleError.mockRestore()
  })
})
