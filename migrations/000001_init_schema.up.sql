-- Splitwise Database Schema
-- Run this migration to create all tables

-- Enable UUID extension if needed
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- ENUM TYPES
-- ============================================

CREATE TYPE split_status AS ENUM ('PENDING', 'PAID', 'CONFIRMED', 'DISPUTED');
CREATE TYPE settlement_status AS ENUM ('PENDING', 'PAID', 'CONFIRMED', 'REJECTED');
CREATE TYPE member_status AS ENUM ('INVITED', 'JOINED');
CREATE TYPE member_role AS ENUM ('ADMIN', 'MEMBER');

-- ============================================
-- USERS TABLE
-- ============================================

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    avatar_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

-- ============================================
-- GROUPS TABLE
-- ============================================

CREATE TABLE groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    is_temporary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- GROUP MEMBERS TABLE
-- ============================================

CREATE TABLE group_members (
    id SERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status member_status DEFAULT 'INVITED',
    role member_role DEFAULT 'MEMBER',
    joined_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure a user isn't in the same group twice
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_group_members_group_id ON group_members(group_id);
CREATE INDEX idx_group_members_user_id ON group_members(user_id);

-- ============================================
-- EXPENSES TABLE
-- ============================================

CREATE TABLE expenses (
    id SERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    payer_id INTEGER NOT NULL REFERENCES users(id),
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    image_url VARCHAR(500),
    split_type VARCHAR(20) NOT NULL DEFAULT 'EVEN', -- EVEN, PERCENTAGE, EXACT
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_expenses_group_id ON expenses(group_id);
CREATE INDEX idx_expenses_payer_id ON expenses(payer_id);
CREATE INDEX idx_expenses_created_at ON expenses(created_at DESC);

-- ============================================
-- SETTLEMENTS TABLE
-- ============================================

CREATE TABLE settlements (
    id SERIAL PRIMARY KEY,
    payer_id INTEGER NOT NULL REFERENCES users(id),
    receiver_id INTEGER NOT NULL REFERENCES users(id),
    amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    currency_code VARCHAR(3) DEFAULT 'SAR',
    status settlement_status DEFAULT 'PENDING',
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Payer and receiver must be different
    CHECK (payer_id != receiver_id)
);

CREATE INDEX idx_settlements_payer_id ON settlements(payer_id);
CREATE INDEX idx_settlements_receiver_id ON settlements(receiver_id);
CREATE INDEX idx_settlements_status ON settlements(status);

-- ============================================
-- SPLITS TABLE
-- ============================================

CREATE TABLE splits (
    id SERIAL PRIMARY KEY,
    expense_id INTEGER NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    borrower_id INTEGER NOT NULL REFERENCES users(id),
    amount_owed DECIMAL(10,2) NOT NULL CHECK (amount_owed >= 0),
    status split_status DEFAULT 'PENDING',
    dispute_reason TEXT,
    settlement_id INTEGER REFERENCES settlements(id) ON DELETE SET NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_splits_expense_id ON splits(expense_id);
CREATE INDEX idx_splits_borrower_id ON splits(borrower_id);
CREATE INDEX idx_splits_status ON splits(status);
CREATE INDEX idx_splits_settlement_id ON splits(settlement_id);

-- ============================================
-- NOTIFICATIONS TABLE
-- ============================================

CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message VARCHAR(500) NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    related_entity_type VARCHAR(50), -- e.g., 'SETTLEMENT', 'EXPENSE', 'GROUP', 'SPLIT'
    related_entity_id INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient_id ON notifications(recipient_id);
CREATE INDEX idx_notifications_is_read ON notifications(recipient_id, is_read);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- ============================================
-- SEED DATA (Optional - for development)
-- ============================================

-- Insert test users
INSERT INTO users (username, email) VALUES 
    ('john_doe', 'john@example.com'),
    ('jane_smith', 'jane@example.com'),
    ('bob_wilson', 'bob@example.com');
