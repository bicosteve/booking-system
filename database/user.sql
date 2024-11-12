CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL, 
    phone_number VARCHAR(11) NOT NULL, 
    hashed_password VARCHAR(100) NOT NULL, 
    is_seller ENUM('YES','NO') DEFAULT 'NO',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_id ON users(id);