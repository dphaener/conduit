-- Auto-generated migration
-- Generated at: 2025-11-01T17:40:18-07:00

-- Add resource: User
CREATE TABLE IF NOT EXISTS "user" (
  "id" UUID NOT NULL,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL,
  "email" VARCHAR(255) NOT NULL,
  "name" VARCHAR(255) NOT NULL
);

