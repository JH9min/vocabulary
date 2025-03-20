CREATE DATABASE IF NOT EXISTS vocabulary_db;
USE vocabulary_db;
-- 使用者
CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- 單字
CREATE TABLE IF NOT EXISTS vocabularies (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    word VARCHAR(100) NOT NULL,
    status ENUM('active', 'removed') NOT NULL DEFAULT 'active',
    tested BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE KEY unique_user_word (user_id, word)
);
-- 單字定義
CREATE TABLE IF NOT EXISTS vocabulary_definitions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    vocabulary_id BIGINT NOT NULL,
    part_of_speech VARCHAR(50) NOT NULL,
    definition TEXT NOT NULL,
    example TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vocabulary_id) REFERENCES vocabularies(id) ON DELETE CASCADE
);
-- 測試結果
CREATE TABLE IF NOT EXISTS test_results (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    word_id BIGINT NOT NULL,
    correct BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (word_id) REFERENCES vocabularies(id)
);
