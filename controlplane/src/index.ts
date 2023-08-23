import * as process from 'node:process';
import pino from 'pino';

import 'dotenv/config';

import build, { BuildConfig } from './core/build-server.js';
import { envVariables } from './core/env.schema.js';

const {
  LOG_LEVEL,
  PORT,
  HOST,
  ALLOWED_ORIGINS,
  DB_URL,
  DB_CERT_PATH,
  DB_KEY_PATH,
  DB_CA_PATH,
  DEBUG_SQL,
  CLICKHOUSE_DSN,
  AUTH_REDIRECT_URI,
  WEB_BASE_URL,
  AUTH_JWT_SECRET,
  KC_REALM,
  KC_CLIENT_ID,
  KC_ADMIN_PASSWORD,
  KC_API_URL,
  KC_FRONTEND_URL,
  KC_ADMIN_USER,
} = envVariables.parse(process.env);

const options: BuildConfig = {
  database: {
    url: DB_URL,
  },
  allowedOrigins: ALLOWED_ORIGINS,
  production: process.env.NODE_ENV === 'production',
  clickhouseDsn: CLICKHOUSE_DSN,
  logger: {
    enabled: true,
    level: LOG_LEVEL as pino.LevelWithSilent,
  },
  keycloak: {
    realm: KC_REALM,
    clientId: KC_CLIENT_ID,
    adminUser: KC_ADMIN_USER,
    adminPassword: KC_ADMIN_PASSWORD,
    apiUrl: KC_API_URL,
    frontendUrl: KC_FRONTEND_URL,
  },
  auth: {
    redirectUri: AUTH_REDIRECT_URI,
    secret: AUTH_JWT_SECRET,
    webBaseUrl: WEB_BASE_URL,
    webErrorPath: '/auth/error',
  },
  debugSQL: DEBUG_SQL,
};

if (DB_CERT_PATH || DB_KEY_PATH || DB_CA_PATH) {
  options.database.ssl = {
    certPath: DB_CERT_PATH,
    keyPath: DB_KEY_PATH,
    caPath: DB_CA_PATH,
  };
}

const app = await build(options);

const addr = await app.listen({
  host: HOST,
  port: PORT,
});
