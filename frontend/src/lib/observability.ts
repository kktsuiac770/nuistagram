export interface LogContext {
  [key: string]: unknown
}

type LogLevel = 'debug' | 'info' | 'warn' | 'error'

interface LoggerConfig {
  service?: string
  environment?: string
  enableConsole?: boolean
}

class Logger {
  private service: string
  private environment: string
  private enableConsole: boolean

  constructor(config: LoggerConfig = {}) {
    this.service = config.service || 'nuistagram-frontend'
    this.environment = config.environment || import.meta.env.MODE || 'development'
    this.enableConsole = config.enableConsole ?? true
  }

  private formatMessage(level: LogLevel, message: string, context?: LogContext): string {
    const timestamp = new Date().toISOString()
    const ctxStr = context ? ` ${JSON.stringify(context)}` : ''
    return `[${timestamp}] ${level.toUpperCase()} [${this.service}] [${this.environment}] ${message}${ctxStr}`
  }

  log(level: LogLevel, message: string, context?: LogContext): void {
    if (!this.enableConsole) return

    const formatted = this.formatMessage(level, message, context)
    switch (level) {
      case 'debug':
        console.debug(formatted)
        break
      case 'info':
        console.info(formatted)
        break
      case 'warn':
        console.warn(formatted)
        break
      case 'error':
        console.error(formatted)
        break
    }
  }

  debug(message: string, context?: LogContext): void {
    this.log('debug', message, context)
  }

  info(message: string, context?: LogContext): void {
    this.log('info', message, context)
  }

  warn(message: string, context?: LogContext): void {
    this.log('warn', message, context)
  }

  error(message: string, error?: Error | unknown, context?: LogContext): void {
    const errorContext: LogContext = { ...context }
    if (error instanceof Error) {
      errorContext.error_name = error.name
      errorContext.error_message = error.message
      errorContext.error_stack = error.stack
    } else if (error) {
      errorContext.error = String(error)
    }
    this.log('error', message, errorContext)
  }

  withContext(defaultContext: LogContext): ContextualLogger {
    return new ContextualLogger(this, defaultContext)
  }
}

class ContextualLogger {
  private logger: Logger
  private defaultContext: LogContext

  constructor(logger: Logger, defaultContext: LogContext) {
    this.logger = logger
    this.defaultContext = defaultContext
  }

  private mergeContext(context?: LogContext): LogContext | undefined {
    if (!context) return this.defaultContext
    return { ...this.defaultContext, ...context }
  }

  debug(message: string, context?: LogContext): void {
    this.logger.debug(message, this.mergeContext(context))
  }

  info(message: string, context?: LogContext): void {
    this.logger.info(message, this.mergeContext(context))
  }

  warn(message: string, context?: LogContext): void {
    this.logger.warn(message, this.mergeContext(context))
  }

  error(message: string, error?: Error | unknown, context?: LogContext): void {
    this.logger.error(message, error, this.mergeContext(context))
  }
}

export const logger = new Logger()

export function captureException(error: Error, context?: LogContext): void {
  logger.error('Uncaught exception', error, context)
}

export function captureMessage(message: string, level: LogLevel = 'error', context?: LogContext): void {
  logger.log(level, message, context)
}

export function initErrorTracking(): void {
  window.onerror = (message, source, lineno, colno, error) => {
    captureException(error || new Error(String(message)), {
      source,
      lineno,
      colno,
    })
    return false
  }

  window.onunhandledrejection = (event) => {
    const error = event.reason instanceof Error
      ? event.reason
      : new Error(String(event.reason))
    captureException(error, { type: 'unhandledrejection' })
  }
}
