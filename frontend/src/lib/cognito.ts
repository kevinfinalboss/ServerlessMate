import { AuthenticationDetails, CognitoUser, CognitoUserPool } from 'amazon-cognito-identity-js'
import type { TranslationKey } from './i18n'

const CLIENT_ID = import.meta.env.VITE_COGNITO_CLIENT_ID as string | undefined
const USER_POOL_ID = import.meta.env.VITE_COGNITO_USER_POOL_ID as string | undefined
const REGION = import.meta.env.VITE_COGNITO_REGION as string | undefined

function getUserPool(): CognitoUserPool {
  if (!CLIENT_ID || !USER_POOL_ID) {
    throw new CognitoError('ConfigurationError', 'VITE_COGNITO_CLIENT_ID / VITE_COGNITO_USER_POOL_ID not configured')
  }
  return new CognitoUserPool({ UserPoolId: USER_POOL_ID, ClientId: CLIENT_ID })
}

export class CognitoError extends Error {
  type: string

  constructor(type: string, message: string) {
    super(message)
    this.type = type
  }
}

async function cognitoRequest<T>(action: string, body: Record<string, unknown>): Promise<T> {
  if (!CLIENT_ID || !REGION) {
    throw new CognitoError('ConfigurationError', 'VITE_COGNITO_CLIENT_ID / VITE_COGNITO_REGION not configured')
  }

  const res = await fetch(`https://cognito-idp.${REGION}.amazonaws.com/`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-amz-json-1.1',
      'X-Amz-Target': `AWSCognitoIdentityProviderService.${action}`,
    },
    body: JSON.stringify({ ClientId: CLIENT_ID, ...body }),
  })

  const data = await res.json()
  if (!res.ok) {
    const type = (data.__type as string | undefined)?.split('#').pop() ?? 'UnknownError'
    throw new CognitoError(type, (data.message as string | undefined) ?? type)
  }
  return data as T
}

export async function signUp(email: string, password: string, username: string): Promise<{ confirmed: boolean }> {
  const out = await cognitoRequest<{ UserConfirmed: boolean }>('SignUp', {
    Username: email,
    Password: password,
    UserAttributes: [
      { Name: 'email', Value: email },
      { Name: 'custom:username', Value: username },
    ],
  })
  return { confirmed: out.UserConfirmed }
}

export async function confirmSignUp(email: string, code: string): Promise<void> {
  await cognitoRequest('ConfirmSignUp', { Username: email, ConfirmationCode: code })
}

export async function resendConfirmationCode(email: string): Promise<void> {
  await cognitoRequest('ResendConfirmationCode', { Username: email })
}

export function signIn(email: string, password: string): Promise<{ idToken: string }> {
  return new Promise((resolve, reject) => {
    const user = new CognitoUser({ Username: email, Pool: getUserPool() })
    const authDetails = new AuthenticationDetails({ Username: email, Password: password })

    user.authenticateUser(authDetails, {
      onSuccess: (session) => resolve({ idToken: session.getIdToken().getJwtToken() }),
      onFailure: (err: { code?: string; message?: string }) => {
        reject(new CognitoError(err.code ?? 'UnknownError', err.message ?? 'Sign in failed'))
      },
    })
  })
}

const ERROR_KEYS: Record<string, TranslationKey> = {
  UsernameExistsException: 'auth.errors.usernameExists',
  NotAuthorizedException: 'auth.errors.notAuthorized',
  UserNotFoundException: 'auth.errors.userNotFound',
  UserNotConfirmedException: 'auth.errors.userNotConfirmed',
  CodeMismatchException: 'auth.errors.codeMismatch',
  ExpiredCodeException: 'auth.errors.codeExpired',
  InvalidPasswordException: 'auth.errors.invalidPassword',
  LimitExceededException: 'auth.errors.tooManyRequests',
  TooManyRequestsException: 'auth.errors.tooManyRequests',
}

export function mapCognitoError(err: unknown): TranslationKey {
  if (err instanceof CognitoError) {
    return ERROR_KEYS[err.type] ?? 'auth.errors.generic'
  }
  return 'auth.errors.generic'
}
