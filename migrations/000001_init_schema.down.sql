-- Rollback migration: Drop all tables

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS splits;
DROP TABLE IF EXISTS settlements;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS split_status;
DROP TYPE IF EXISTS settlement_status;
DROP TYPE IF EXISTS member_status;
DROP TYPE IF EXISTS member_role;
