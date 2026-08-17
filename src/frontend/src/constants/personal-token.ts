export const PERSONAL_TOKEN_REALMS = ['blueking', 'bk-devops', 'bk-gpu'] as const;

export type PersonalTokenRealm = typeof PERSONAL_TOKEN_REALMS[number];

export const DEFAULT_PERSONAL_TOKEN_REALM: PersonalTokenRealm = 'blueking';

export const isPersonalTokenRealm = (value: unknown): value is PersonalTokenRealm =>
  typeof value === 'string' && PERSONAL_TOKEN_REALMS.includes(value as PersonalTokenRealm);
