# Hotel Booking System

## Overview

This is a hotel booking system built to manage user registrations, reservations, and payments. The system supports role-based access control, distinguishing between admin and ordinary users. Admin users have elevated privileges such as managing bookings and overseeing system operations, while ordinary users can register, make bookings, and manage their reservations.

The system features seamless integration with Stripe for secure online payments. It also incorporates SMS and email services to notify users upon registration and booking confirmations. Additional features include a password reset mechanism to enhance account security.

## Features

- **User Registration and Authentication**
- **Role-Based Access Control (Admin & User)**
- **Stripe Payment Integration**
- **SMS & Email Notifications**
- **Password Reset Functionality**
- **Swagger API Documentation**

## Technologies Used

- **Golang** – Backend logic and routing
- **Kafka** – Event-driven message processing
- **Stripe** – Payment processing
- **Redis** – Caching and session management
- **SMS & Email Integration** – Notifications and alerts

## API Documentation

Swagger UI is available at:

```bash
    http://localhost:7001/swagger/index.html
    http://localhost:7002/swagger/index.html
```

### 🟢 Public Routes

| Method | Endpoint             | Description                        |
| ------ | -------------------- | ---------------------------------- |
| POST   | `/api/user/register` | Register a new user                |
| POST   | `/api/user/login`    | Log in an existing user            |
| GET    | `/api/user/rooms`    | Retrieve a list of available rooms |

### 🔒 Private User Routes (Authentication Required)

| Method | Endpoint                          | Description                     |
| ------ | --------------------------------- | ------------------------------- |
| GET    | `/api/user/me`                    | Get user profile                |
| POST   | `/api/user/reset`                 | Request password reset token    |
| POST   | `/api/user/password-reset`        | Reset user password using token |
| POST   | `/api/user/book`                  | Create a new booking            |
| GET    | `/api/user/book/verify/{room_id}` | Verify a room booking           |
| GET    | `/api/user/book/{room_id}`        | Get booking details for a room  |
| GET    | `/api/user/book/all`              | Get all user bookings           |
| PUT    | `/api/user/book/{booking_id}`     | Update a booking                |

### 🔐 Admin Routes (Admin Authentication Required)

| Method | Endpoint                                 | Description               |
| ------ | ---------------------------------------- | ------------------------- |
| POST   | `/api/admin/rooms`                       | Create a new room         |
| PUT    | `/api/admin/rooms/{room_id}`             | Update room details       |
| DELETE | `/api/admin/rooms/{room_id}`             | Delete a room             |
| GET    | `/api/admin/book/all`                    | Retrieve all bookings     |
| DELETE | `/api/admin/book/{booking_id}/{room_id}` | Delete a specific booking |

## Getting Started

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/your-username/hotel-booking-system.git
   cd hotel-booking-system
   ```

2. **Environment Setup:**

- create a .toml file to hold hold the keys and secrets for stripe, kafka, redis,email,sms providers, jwt secret
- Set up mysql tables by running the file on ./files/

3. **Install Dependancies**

```bash
    go mod tidy
```

4. **Run the application**

```bash
    make servers
```

5. **Access the endpoints**

```bash
    http://localhost:7001/swagger/index.html
    http://localhost:7002/swagger/index.html
```

6. **Prod environment**

- This application runs on gcp vm with nginx as proxy.

7. **Future Plans**

- Increase the test coverage
- Add docker containers for the application
