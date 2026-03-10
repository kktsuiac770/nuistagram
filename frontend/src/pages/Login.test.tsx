import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import Login from './Login'
import { AuthProvider } from '../contexts/Auth'
import * as apiModule from '../lib/api'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

function renderLogin() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <Login />
      </AuthProvider>
    </MemoryRouter>
  )
}

describe('Login', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(apiModule.api, 'getMe').mockRejectedValue(new Error('Unauthorized'))
  })

  it('should render login form', async () => {
    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    })
    expect(screen.getByPlaceholderText('Password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument()
  })

  it('should show NUIstagram title', async () => {
    renderLogin()

    await waitFor(() => {
      expect(screen.getByText('NUIstagram')).toBeInTheDocument()
    })
  })

  it('should show sign up link', async () => {
    renderLogin()

    await waitFor(() => {
      expect(screen.getByText("Don't have an account?")).toBeInTheDocument()
    })
    expect(screen.getByText('Sign up')).toBeInTheDocument()
  })

  it('should update username input on change', async () => {
    const user = userEvent.setup()
    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    })

    const usernameInput = screen.getByPlaceholderText('Username')
    await user.type(usernameInput, 'testuser')

    expect(usernameInput).toHaveValue('testuser')
  })

  it('should update password input on change', async () => {
    const user = userEvent.setup()
    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Password')).toBeInTheDocument()
    })

    const passwordInput = screen.getByPlaceholderText('Password')
    await user.type(passwordInput, 'Password123')

    expect(passwordInput).toHaveValue('Password123')
  })

  it('should call login on form submit', async () => {
    const user = userEvent.setup()
    vi.spyOn(apiModule.api, 'login').mockResolvedValue({ access_token: "fake_access_token", refresh_token: "fake_refresh_token", expires_in: 3600, token_type: "Bearer" })
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'testuser' })

    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByPlaceholderText('Username'), 'testuser')
    await user.type(screen.getByPlaceholderText('Password'), 'Password123')

    await act(async () => {
      await user.click(screen.getByRole('button', { name: 'Log in' }))
    })

    expect(apiModule.api.login).toHaveBeenCalledWith('testuser', 'Password123')
  })

  it('should show loading state during login', async () => {
    const user = userEvent.setup()
    vi.spyOn(apiModule.api, 'login').mockImplementation(() => new Promise(() => { }))

    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByPlaceholderText('Username'), 'testuser')
    await user.type(screen.getByPlaceholderText('Password'), 'Password123')

    await act(async () => {
      await user.click(screen.getByRole('button', { name: 'Log in' }))
    })

    expect(screen.getByRole('button', { name: 'Logging in...' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Logging in...' })).toBeDisabled()
  })

  it('should show error message on login failure', async () => {
    const user = userEvent.setup()
    vi.spyOn(apiModule.api, 'login').mockRejectedValue(new Error('Invalid credentials'))

    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByPlaceholderText('Username'), 'wronguser')
    await user.type(screen.getByPlaceholderText('Password'), 'wrongpass')

    await act(async () => {
      await user.click(screen.getByRole('button', { name: 'Log in' }))
    })

    await waitFor(() => {
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument()
    })
  })

  it('should navigate to home on successful login', async () => {
    const user = userEvent.setup()
    vi.spyOn(apiModule.api, 'login').mockResolvedValue({ access_token: "fake_access_token", refresh_token: "fake_refresh_token", expires_in: 3600, token_type: "Bearer" })
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'testuser' })

    renderLogin()

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    })

    await user.type(screen.getByPlaceholderText('Username'), 'testuser')
    await user.type(screen.getByPlaceholderText('Password'), 'Password123')

    await act(async () => {
      await user.click(screen.getByRole('button', { name: 'Log in' }))
    })

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/')
    })
  })
})
