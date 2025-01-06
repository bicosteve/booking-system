CREATE TABLE user (
    user_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL, 
    phone_number VARCHAR(11) NOT NULL, 
    isVender ENUM('YES','NO') NOT NULL DEFAULT 'NO',
    hashed_password VARCHAR(100) NOT NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_id ON user(user_id);


CREATE TABLE room (
    room_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    cost DECIMAL (10,2) NOT NULL, 
    status ENUM('BOOKED','VACANT') NOT NULL DEFAULT 'VACANT',
    vender_id BIGINT NOT NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (vender_id) REFERENCES user(user_id)
);

CREATE INDEX idx_room_id ON room(room_id);

CREATE TABLE booking (
    booking_id BIGINT PRIMARY KEY AUTO_INCREMENT, 
    days BIGINT NOT NULL, 
    user_id BIGINT NOT NULL, 
    room_id BIGINT NOT NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(user_id),
    FOREIGN KEY (room_id) REFERENCES room(room_id)
);

CREATE INDEX idx_booking_id ON booking(booking_id);

CREATE TABLE transaction(
    transaction_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    room_id BIGINT NOT NULL, 
    user_id BIGINT NOT NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(user_id),
    FOREIGN KEY (room_id) REFERENCES room(room_id)
);

CREATE INDEX idx_transaction_id ON transaction(transaction_id);