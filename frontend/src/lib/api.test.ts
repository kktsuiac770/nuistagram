import { describe, it, expect } from 'vitest'
import { validatePassword, passwordRequirements, getAvatarUrl } from './api'

describe('validatePassword', () => {
  it('should return null for valid password', () => {
    expect(validatePassword('Password123')).toBeNull()
  })

  it('should return error for password shorter than minimum length', () => {
    const result = validatePassword('Pass1')
    expect(result).toBe(`Password must be at least ${passwordRequirements.minLength} characters long`)
  })

  it('should return error for password without uppercase letter', () => {
    const result = validatePassword('password123')
    expect(result).toBe('Password must contain at least one uppercase letter')
  })

  it('should return error for password without lowercase letter', () => {
    const result = validatePassword('PASSWORD123')
    expect(result).toBe('Password must contain at least one lowercase letter')
  })

  it('should return error for password without digit', () => {
    const result = validatePassword('Passwordddd')
    expect(result).toBe('Password must contain at least one digit')
  })

  it('should accept password with exactly minimum length', () => {
    expect(validatePassword('Passwor1')).toBeNull()
  })
})

describe('getAvatarUrl', () => {
  it('should return empty string for undefined avatar', () => {
    expect(getAvatarUrl()).toBe('')
  })

  it('should return empty string for empty string avatar', () => {
    expect(getAvatarUrl('')).toBe('')
  })

  it('should return URL as-is for http URLs', () => {
    expect(getAvatarUrl('http://example.com/avatar.jpg')).toBe('http://example.com/avatar.jpg')
  })

  it('should return URL as-is for https URLs', () => {
    expect(getAvatarUrl('https://example.com/avatar.jpg')).toBe('https://example.com/avatar.jpg')
  })

  it('should prepend static path for relative avatar paths', () => {
    expect(getAvatarUrl('avatar.jpg')).toBe('/static/uploads/avatar.jpg')
  })
})
